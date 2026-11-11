package app

import (
	"crypto/sha256"
	"math"
	"sync"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	crossvmtypes "github.com/qorechain/qorechain-core/x/crossvm/types"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	"github.com/qorechain/qorechain-core/x/wasmfilter"
)

// ErrBlockWasmBudget is the refusal of a CosmWasm VM call once the block's
// wasm execution budget, or the calling transaction's share of it, is spent
// (wasm-block-gas-budget).
var ErrBlockWasmBudget = errorsmod.Register(wasmfilter.ModuleName, 3, "block wasm gas budget exhausted")

// blockWasmBudget is the per-block CosmWasm execution budget
// (wasm-block-gas-budget), in VM gas.
//
// Why the count is kept here and not in a store. It has to include the work
// of transactions that fail (an out-of-gas loop is exactly the expensive
// case), and every store write a transaction makes, a transient one
// included, is dropped when the transaction fails. So the count lives in the
// application for the duration of one block's execution, and is derived from
// that execution alone:
//
//   - it is reset at the start of every block, in PreBlock, from the block's
//     own state (switch and parameters), before any transaction runs; a
//     block executed again (a node restarting, an aborted and repeated
//     execution) starts again from the full budget;
//   - only VM calls made while executing the block are counted (exec mode
//     finalize, at the block's height); CheckTx, simulations and queries,
//     which run beside block execution, are neither counted nor limited;
//   - block execution is sequential, so every node counts the same calls in
//     the same order and refuses the same one.
//
// Three kinds of call are told apart:
//
//   - a transaction's (the context carries its bytes): limited by what is
//     left of the block budget and by the transaction's own share of it
//     (max_tx_wasm_share_bps), and refused when either is spent;
//   - the end-of-block async queue's (crossvmtypes.IsQueueExecution):
//     limited by what is left; the queue itself leaves a call pending when
//     the budget cannot cover the call's whole prepaid gas (queueWasmFits);
//   - any other call outside a transaction (a governance proposal executed
//     at the end of the block, on the block's infinite gas meter): limited to
//     max_block_level_wasm_gas, so it ends with out of gas instead of running
//     forever, and counted, but never refused because transactions spent the
//     budget (that would let anyone make a proposal fail).
type blockWasmBudget struct {
	mu         sync.Mutex
	on         bool
	height     int64
	remaining  uint64
	txCap      uint64
	blockLevel uint64
	tx         [32]byte
	txUsed     uint64
}

// saturatingVMGas converts SDK gas to VM gas at multiplier m without
// overflowing.
func saturatingVMGas(sdkGas, m uint64) uint64 {
	if m != 0 && sdkGas > math.MaxUint64/m {
		return math.MaxUint64
	}
	return sdkGas * m
}

// reset starts a block. Limits are in VM gas.
func (b *blockWasmBudget) reset(height int64, on bool, limit, txCap, blockLevel uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.on, b.height, b.remaining, b.txCap, b.blockLevel = on, height, limit, txCap, blockLevel
	b.tx, b.txUsed = [32]byte{}, 0
}

func (b *blockWasmBudget) counts(ctx sdk.Context) bool {
	return b.on && ctx.ExecMode() == sdk.ExecModeFinalize && ctx.BlockHeight() == b.height
}

func min64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Limit gives a VM call at most what its kind may still use.
func (b *blockWasmBudget) Limit(ctx sdk.Context, gasLimit uint64) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return gasLimit, nil
	}
	switch txBytes := ctx.TxBytes(); {
	case len(txBytes) > 0:
		if b.remaining == 0 {
			return 0, ErrBlockWasmBudget.Wrap("this block's CosmWasm execution budget is spent; retry in the next block")
		}
		if key := sha256.Sum256(txBytes); key != b.tx {
			b.tx, b.txUsed = key, 0
		}
		if b.txUsed >= b.txCap {
			return 0, ErrBlockWasmBudget.Wrap("this transaction has used its share of the block's CosmWasm execution budget")
		}
		return min64(gasLimit, min64(b.remaining, b.txCap-b.txUsed)), nil
	case crossvmtypes.IsQueueExecution(ctx):
		if b.remaining == 0 {
			return 0, ErrBlockWasmBudget.Wrap("this block's CosmWasm execution budget is spent")
		}
		return min64(gasLimit, b.remaining), nil
	default:
		return min64(gasLimit, b.blockLevel), nil
	}
}

// Used takes a VM call's gas from the budget (and from the transaction's
// share).
func (b *blockWasmBudget) Used(ctx sdk.Context, gasUsed uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return
	}
	if txBytes := ctx.TxBytes(); len(txBytes) > 0 && sha256.Sum256(txBytes) == b.tx {
		b.txUsed += min64(gasUsed, math.MaxUint64-b.txUsed)
	}
	if gasUsed >= b.remaining {
		b.remaining = 0
		return
	}
	b.remaining -= gasUsed
}

// Remaining is what is left, in VM gas. For tests and tools.
func (b *blockWasmBudget) Remaining() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.remaining
}

// fits reports whether vmGas of block execution can still be covered (always
// true when the budget is off or ctx is not block execution).
func (b *blockWasmBudget) fits(ctx sdk.Context, vmGas uint64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.counts(ctx) || b.remaining >= vmGas
}

// wasmBlockBudget returns the app's one budget.
func (app *QoreChainApp) wasmBlockBudget() *blockWasmBudget {
	if app.blockBudget == nil {
		app.blockBudget = &blockWasmBudget{}
	}
	return app.blockBudget
}

// resetWasmBlockBudget starts the block's budget from ctx's state. It runs
// after the gas multiplier is set, so the budget converts to VM gas at the
// block's price, saturating rather than overflowing. Reads are free of gas.
func (app *QoreChainApp) resetWasmBlockBudget(ctx sdk.Context) {
	on := app.PQCKeeper != nil && app.CrossVMKeeper != nil &&
		app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureWasmBlockGasBudget)
	var limit, txCap, blockLevel uint64
	if on {
		p := app.CrossVMKeeper.GetParams(ctx)
		m := app.wasmGasRegister().Multiplier()
		limit = saturatingVMGas(p.BlockWasmGasLimit(), m)
		txCap = limit / 10_000 * uint64(p.TxWasmShareBps())
		blockLevel = saturatingVMGas(p.BlockLevelWasmGasLimit(), m)
	}
	app.wasmBlockBudget().reset(ctx.BlockHeight(), on, limit, txCap, blockLevel)
}

// queueWasmFits reports whether the block's wasm budget can still cover a
// queued call's whole prepaid gas; the queue leaves the call pending for a
// later block when it cannot.
func (app *QoreChainApp) queueWasmFits(ctx sdk.Context, sdkGas uint64) bool {
	return app.wasmBlockBudget().fits(ctx, saturatingVMGas(sdkGas, app.wasmGasRegister().Multiplier()))
}
