package app

import (
	"errors"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

const (
	// DefaultPrepareProposalBudget is how long PrepareProposal may spend
	// selecting transactions before it returns what it already has. A third of
	// the ~3 s block interval, so a large backlog can never push the proposer
	// past its propose timeout.
	DefaultPrepareProposalBudget = time.Second

	// DefaultPrepareProposalMaxSkips is how many candidates in a row may be
	// skipped for not fitting in the remaining block space before selection
	// stops.
	DefaultPrepareProposalMaxSkips = 64

	// minTxGas is the smallest gas a transaction can want. Once less than this
	// is left in the block nothing else can fit.
	minTxGas = 21_000
)

// proposalLimits tunes the PrepareProposal handler. The zero value of a field
// disables that limit.
type proposalLimits struct {
	// Budget is the wall-clock time from handler entry after which no further
	// candidate is verified.
	Budget time.Duration
	// MaxSkips is the number of consecutive non-fitting candidates after which
	// selection stops.
	MaxSkips int
}

// prepareProposalHandler is the SDK default PrepareProposal handler (cosmos-sdk
// v0.53 baseapp/abci_utils.go) with the fit check moved ahead of verification
// (QSR-2026-0029).
//
// The default handler runs the full ante chain, including the ML-DSA-87 FFI
// verify, on every mempool candidate and only then asks the selector whether
// the transaction fits. Once the block is nearly full every remaining candidate
// is still verified and then dropped, so the ante cost of a proposal grows with
// the whole backlog instead of with the block. Here a candidate is encoded and
// measured first, and one that cannot fit in the remaining bytes or gas is
// skipped without running the ante and without being treated as invalid; it
// stays in the pool for a later block. Selection also stops after MaxSkips
// non-fitting candidates in a row, when the space left is smaller than any
// candidate seen, or when the time budget runs out.
//
// Everything the handler includes has passed the same PrepareProposalVerifyTx
// the default handler uses, in the same order and against the same proposal
// state, and the byte and gas totals never exceed the limits, so the output is
// always accepted by the unchanged default ProcessProposal handler. The change
// is local to the proposer and needs no coordination.
func prepareProposalHandler(
	mp sdkmempool.Mempool,
	txVerifier baseapp.ProposalTxVerifier,
	signerAdapter sdkmempool.SignerExtractionAdapter,
	limits proposalLimits,
	logger log.Logger,
) sdk.PrepareProposalHandler {
	if signerAdapter == nil {
		signerAdapter = sdkmempool.NewDefaultSignerExtractionAdapter()
	}
	fallback := baseapp.NewDefaultProposalHandler(mp, txVerifier).PrepareProposalHandler()

	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		// With no app-side mempool the default handler only re-emits the
		// consensus engine's own list and never runs the ante, so there is
		// nothing to bound.
		if _, isNoOp := mp.(sdkmempool.NoOpMempool); mp == nil || isNoOp {
			return fallback(ctx, req)
		}

		start := time.Now()

		// A non-positive MaxGas means no block gas limit, which is how the
		// default ProcessProposal handler reads it too.
		var maxBlockGas uint64
		if b := ctx.ConsensusParams().Block; b != nil && b.MaxGas > 0 {
			maxBlockGas = uint64(b.MaxGas)
		}
		var maxTxBytes uint64
		if req.MaxTxBytes > 0 {
			maxTxBytes = uint64(req.MaxTxBytes)
		}

		var (
			selected     [][]byte
			totalBytes   uint64
			totalGas     uint64
			minSeenBytes uint64 // smallest candidate seen so far that fits an empty block
			skipsInRow   int
			verified     int
			stopReason   string

			resError   error
			invalidTxs []sdk.Tx // removed after the loop to avoid a deadlock
		)
		selectedTxsSignersSeqs := make(map[string]uint64)

		fits := func(size, gas uint64) bool {
			if size > maxTxBytes-totalBytes {
				return false
			}
			return maxBlockGas == 0 || gas <= maxBlockGas-totalGas
		}
		full := func() bool {
			if totalBytes >= maxTxBytes || (minSeenBytes > 0 && maxTxBytes-totalBytes < minSeenBytes) {
				return true
			}
			return maxBlockGas > 0 && maxBlockGas-totalGas < minTxGas
		}

		sdkmempool.SelectBy(ctx, mp, req.Txs, func(memTx sdk.Tx) bool {
			if limits.Budget > 0 && time.Since(start) >= limits.Budget {
				stopReason = "time budget"
				return false
			}

			unorderedTx, ok := memTx.(sdk.TxWithUnordered)
			isUnordered := ok && unorderedTx.GetUnordered()
			txSignersSeqs := make(map[string]uint64)

			// Same sequence bookkeeping as the default handler: a signer seen
			// earlier in this proposal must continue at seq+1.
			if !isUnordered {
				signerData, err := signerAdapter.GetSigners(memTx)
				if err != nil {
					resError = err
					return false
				}
				for _, signer := range signerData {
					seq, seen := selectedTxsSignersSeqs[signer.Signer.String()]
					if seen && seq+1 != signer.Sequence {
						return true
					}
					txSignersSeqs[signer.Signer.String()] = signer.Sequence
				}
			}

			// A candidate left out of this block must also hold back its
			// signers' later transactions. Recording seq-1 (as the default
			// handler does for a verified but unselected tx) makes the next
			// nonce fail the continuity check above without being verified,
			// instead of being verified against a state that never saw this
			// one and removed as invalid.
			holdBackSigners := func() {
				for sender, seq := range txSignersSeqs {
					if _, seen := selectedTxsSignersSeqs[sender]; !seen {
						selectedTxsSignersSeqs[sender] = seq - 1
					}
				}
			}

			var gasWanted uint64
			if gasTx, ok := memTx.(baseapp.GasTx); ok {
				gasWanted = gasTx.GetGas()
			}

			txBz, err := txVerifier.TxEncode(memTx)
			if err != nil {
				// PrepareProposalVerifyTx would fail on the same encode.
				invalidTxs = append(invalidTxs, memTx)
				return true
			}
			size := uint64(cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{txBz}))
			// A tx larger than the whole block says nothing about what else
			// could still fit, so it does not count towards the smallest seen.
			if size <= maxTxBytes && (minSeenBytes == 0 || size < minSeenBytes) {
				minSeenBytes = size
			}

			if !fits(size, gasWanted) {
				holdBackSigners()
				skipsInRow++
				if limits.MaxSkips > 0 && skipsInRow >= limits.MaxSkips {
					stopReason = "consecutive skips"
					return false
				}
				if full() {
					stopReason = "block full"
					return false
				}
				return true
			}

			verified++
			txBz, err = txVerifier.PrepareProposalVerifyTx(memTx)
			if err != nil {
				invalidTxs = append(invalidTxs, memTx)
				return true
			}

			// The verifier re-encodes; deterministic encoding gives the same
			// bytes, but never include something that does not fit.
			size = uint64(cmttypes.ComputeProtoSizeForTxs([]cmttypes.Tx{txBz}))
			if !fits(size, gasWanted) {
				holdBackSigners()
				skipsInRow++
				return !full()
			}

			selected = append(selected, txBz)
			totalBytes += size
			totalGas += gasWanted
			skipsInRow = 0
			if !isUnordered {
				for sender, seq := range txSignersSeqs {
					selectedTxsSignersSeqs[sender] = seq
				}
			}

			if full() {
				stopReason = "block full"
				return false
			}
			return true
		})

		if resError != nil {
			return nil, resError
		}

		for _, tx := range invalidTxs {
			if err := mp.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
				return nil, err
			}
		}

		if stopReason == "time budget" || stopReason == "consecutive skips" {
			logger.Info("prepare proposal stopped early",
				"reason", stopReason,
				"height", req.Height,
				"selected", len(selected),
				"verified", verified,
				"invalid", len(invalidTxs),
				"elapsed", time.Since(start),
			)
		} else {
			logger.Debug("prepare proposal",
				"height", req.Height,
				"selected", len(selected),
				"verified", verified,
				"invalid", len(invalidTxs),
				"elapsed", time.Since(start),
			)
		}

		return &abci.ResponsePrepareProposal{Txs: selected}, nil
	}
}
