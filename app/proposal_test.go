package app

import (
	"math/rand"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

func proposalCtx(maxGas int64) sdk.Context {
	return sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger()).
		WithConsensusParams(cmtproto.ConsensusParams{Block: &cmtproto.BlockParams{MaxGas: maxGas}})
}

func fillPool(t *testing.T, mp sdkmempool.Mempool, txs []*memTx) {
	t.Helper()
	ctx := proposalCtx(-1)
	for _, tx := range txs {
		if err := mp.Insert(ctx, tx); err != nil {
			t.Fatalf("insert tx %d: %v", tx.id, err)
		}
	}
}

func testLimits() proposalLimits {
	return proposalLimits{Budget: 0, MaxSkips: DefaultPrepareProposalMaxSkips}
}

func runPrepare(t *testing.T, mp sdkmempool.Mempool, v *memVerifier, limits proposalLimits, maxBytes, maxGas int64) [][]byte {
	t.Helper()
	h := prepareProposalHandler(mp, v, memTxSigners{}, limits, log.NewNopLogger())
	res, err := h(proposalCtx(maxGas), &abci.RequestPrepareProposal{MaxTxBytes: maxBytes, Height: 2})
	if err != nil {
		t.Fatalf("prepare proposal: %v", err)
	}
	return res.Txs
}

// processAccepts runs the unchanged SDK ProcessProposal handler on a fresh
// state, as every other validator would.
func processAccepts(t *testing.T, mp sdkmempool.Mempool, v *memVerifier, txs [][]byte, maxGas int64) bool {
	t.Helper()
	v.reset()
	h := baseapp.NewDefaultProposalHandler(mp, v).ProcessProposalHandler()
	res, err := h(proposalCtx(maxGas), &abci.RequestProcessProposal{Txs: txs, Height: 2})
	if err != nil {
		t.Fatalf("process proposal: %v", err)
	}
	return res.Status == abci.ResponseProcessProposal_ACCEPT
}

func blockBytes(txs [][]byte) int64 {
	ctxs := make([]cmttypes.Tx, len(txs))
	for i, tx := range txs {
		ctxs[i] = tx
	}
	return cmttypes.ComputeProtoSizeForTxs(ctxs)
}

// The reporter's PoC (QSR-2026-0029): a backlog of large txs from distinct
// signers. The default handler verifies every one of them; this handler only
// verifies what it can include.
func TestPrepareProposalVerifiesOnlyWhatFits(t *testing.T) {
	const n = 5000
	const txSize = 20_000
	const maxBytes = 1_000_000

	var txs []*memTx
	for i := 0; i < n; i++ {
		txs = append(txs, &memTx{id: i, signer: addr(i), gas: 100_000, size: txSize, fee: int64(1 + i%7)})
	}

	// Baseline: the SDK default handler.
	mpDefault := newMemPool(0)
	fillPool(t, mpDefault, txs)
	vDefault := newMemVerifier(txs)
	def := baseapp.NewDefaultProposalHandler(mpDefault, vDefault)
	def.SetSignerExtractionAdapter(memTxSigners{})
	if _, err := def.PrepareProposalHandler()(proposalCtx(-1), &abci.RequestPrepareProposal{MaxTxBytes: maxBytes}); err != nil {
		t.Fatal(err)
	}
	if vDefault.verifyCalls != n {
		t.Fatalf("baseline: expected the default handler to verify all %d candidates, got %d", n, vDefault.verifyCalls)
	}

	mp := newMemPool(0)
	fillPool(t, mp, txs)
	v := newMemVerifier(txs)
	selected := runPrepare(t, mp, v, testLimits(), maxBytes, -1)

	if len(selected) == 0 {
		t.Fatal("nothing selected")
	}
	if v.verifyCalls > len(selected)+DefaultPrepareProposalMaxSkips {
		t.Fatalf("verified %d candidates for %d selected (bound %d)", v.verifyCalls, len(selected), len(selected)+DefaultPrepareProposalMaxSkips)
	}
	if got := blockBytes(selected); got > maxBytes {
		t.Fatalf("block is %d bytes, over %d", got, maxBytes)
	}
	if mp.CountTx() != n {
		t.Fatalf("skipped txs must stay in the pool: have %d, want %d", mp.CountTx(), n)
	}
	if !processAccepts(t, mp, v, selected, -1) {
		t.Fatal("ProcessProposal rejected the proposal")
	}
}

// Gas-bound backlog: candidates that can never fit the remaining gas are
// skipped without verification, and selection stops after MaxSkips of them.
func TestPrepareProposalStopsAfterConsecutiveSkips(t *testing.T) {
	const maxGas = 1_000_000
	txs := []*memTx{{id: 0, signer: addr(0), gas: 900_000, size: 300, fee: 1_000}}
	for i := 1; i <= 5000; i++ {
		txs = append(txs, &memTx{id: i, signer: addr(i), gas: 500_000, size: 300, fee: 10})
	}
	mp := newMemPool(0)
	fillPool(t, mp, txs)
	v := newMemVerifier(txs)

	selected := runPrepare(t, mp, v, testLimits(), 10_000_000, maxGas)
	if len(selected) != 1 {
		t.Fatalf("expected only the first tx, got %d", len(selected))
	}
	if v.verifyCalls != 1 {
		t.Fatalf("expected 1 verification, got %d", v.verifyCalls)
	}
	if !processAccepts(t, mp, v, selected, maxGas) {
		t.Fatal("ProcessProposal rejected the proposal")
	}
}

// A signer's later nonce must not be verified (and removed as invalid) after
// its earlier nonce was skipped for not fitting.
func TestPrepareProposalHoldsBackSignerAfterSkip(t *testing.T) {
	const maxBytes = 10_000
	txs := []*memTx{
		{id: 0, signer: addr(1), seq: 0, gas: 100_000, size: 20_000, fee: 5}, // never fits
		{id: 1, signer: addr(1), seq: 1, gas: 100_000, size: 200, fee: 5},
		{id: 2, signer: addr(2), seq: 0, gas: 100_000, size: 200, fee: 1},
	}
	mp := newMemPool(0)
	fillPool(t, mp, txs)
	v := newMemVerifier(txs)

	selected := runPrepare(t, mp, v, testLimits(), maxBytes, -1)
	if len(selected) != 1 {
		t.Fatalf("expected only signer 2's tx, got %d", len(selected))
	}
	if tx, _ := v.TxDecode(selected[0]); tx.(*memTx).id != 2 {
		t.Fatalf("selected the wrong tx")
	}
	if len(v.rejected) != 0 {
		t.Fatalf("no tx should have been verified and rejected, got %v", v.rejected)
	}
	if mp.CountTx() != 3 {
		t.Fatalf("pool lost txs: %d left", mp.CountTx())
	}
}

func TestPrepareProposalRemovesInvalidTxs(t *testing.T) {
	txs := []*memTx{
		{id: 0, signer: addr(1), gas: 100_000, size: 200, fee: 3, invalid: true},
		{id: 1, signer: addr(2), gas: 100_000, size: 200, fee: 2},
		{id: 2, signer: addr(3), seq: 4, gas: 100_000, size: 200, fee: 1}, // wrong sequence
	}
	mp := newMemPool(0)
	fillPool(t, mp, txs)
	v := newMemVerifier(txs)

	selected := runPrepare(t, mp, v, testLimits(), 1_000_000, -1)
	if len(selected) != 1 {
		t.Fatalf("expected 1 selected, got %d", len(selected))
	}
	// The selected tx is removed at FinalizeBlock, not here; the two invalid
	// ones are removed now.
	if mp.CountTx() != 1 {
		t.Fatalf("expected the invalid txs to be removed, %d left", mp.CountTx())
	}
}

func TestPrepareProposalTimeBudgetReturnsValidPrefix(t *testing.T) {
	var txs []*memTx
	for i := 0; i < 200; i++ {
		txs = append(txs, &memTx{id: i, signer: addr(i % 20), seq: uint64(i / 20), gas: 100_000, size: 300, fee: 1})
	}
	mp := newMemPool(0)
	fillPool(t, mp, txs)
	v := newMemVerifier(txs)
	v.delay = 2 * time.Millisecond

	start := time.Now()
	selected := runPrepare(t, mp, v, proposalLimits{Budget: 30 * time.Millisecond, MaxSkips: 64}, 10_000_000, -1)
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("handler ran %v despite a 30ms budget", elapsed)
	}
	if len(selected) == 0 || len(selected) >= len(txs) {
		t.Fatalf("expected a non-empty strict prefix, got %d of %d", len(selected), len(txs))
	}
	v.delay = 0
	if !processAccepts(t, mp, v, selected, -1) {
		t.Fatal("ProcessProposal rejected the budget-truncated proposal")
	}
	if mp.CountTx() != len(txs) {
		t.Fatalf("budget stop must not remove txs: %d left", mp.CountTx())
	}
}

// Randomised pools with multi-tx signers, nonce gaps, invalid txs and mixed
// sizes and gas: whatever the handler proposes, the unchanged ProcessProposal
// accepts, the limits hold, and only txs the verifier rejected are removed.
func TestPrepareProposalOutputAlwaysAccepted(t *testing.T) {
	r := rand.New(rand.NewSource(29))
	for round := 0; round < 200; round++ {
		var txs []*memTx
		id := 0
		signers := 1 + r.Intn(40)
		for s := 0; s < signers; s++ {
			seq := uint64(0)
			for k := r.Intn(6); k >= 0; k-- {
				if r.Intn(8) == 0 {
					seq++ // nonce gap
				}
				txs = append(txs, &memTx{
					id:      id,
					signer:  addr(s),
					seq:     seq,
					gas:     uint64(21_000 + r.Intn(2_000_000)),
					size:    32 + r.Intn(30_000),
					fee:     int64(1 + r.Intn(10)),
					invalid: r.Intn(15) == 0,
				})
				id++
				seq++
			}
		}
		mp := newMemPool(0)
		fillPool(t, mp, txs)
		v := newMemVerifier(txs)

		maxBytes := int64(1_000 + r.Intn(300_000))
		maxGas := int64(-1)
		if r.Intn(2) == 0 {
			maxGas = int64(100_000 + r.Intn(20_000_000))
		}
		limits := proposalLimits{MaxSkips: r.Intn(10)}

		before := mp.CountTx()
		selected := runPrepare(t, mp, v, limits, maxBytes, maxGas)

		if got := blockBytes(selected); got > maxBytes {
			t.Fatalf("round %d: %d bytes over %d", round, got, maxBytes)
		}
		if maxGas > 0 {
			var gas uint64
			for _, bz := range selected {
				tx, _ := v.TxDecode(bz)
				gas += tx.(*memTx).gas
			}
			if gas > uint64(maxGas) {
				t.Fatalf("round %d: gas %d over %d", round, gas, maxGas)
			}
		}
		if removed := before - mp.CountTx(); removed != len(v.rejected) {
			t.Fatalf("round %d: removed %d txs but the verifier rejected %d", round, removed, len(v.rejected))
		}
		if !processAccepts(t, mp, v, selected, maxGas) {
			t.Fatalf("round %d: ProcessProposal rejected the proposal", round)
		}
	}
}
