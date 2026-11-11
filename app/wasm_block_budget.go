package app

import (
	"sync"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	"github.com/qorechain/qorechain-core/x/wasmfilter"
)

// ErrBlockWasmBudget is the refusal of a CosmWasm VM call once the block's
// wasm execution budget is spent (wasm-block-gas-budget).
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
//     own state (switch and parameter), before any transaction runs; a block
//     executed again (a node restarting, an aborted and repeated execution)
//     starts again from the full budget;
//   - only VM calls made while executing the block's transactions are
//     counted (exec mode finalize, with transaction bytes, at the block's
//     height); CheckTx, simulations and queries, which run beside block
//     execution, and the begin- and end-block work are neither counted nor
//     limited;
//   - the transactions of a block execute one after another, in the block's
//     order, so every node counts the same calls in the same order and
//     refuses the same one.
type blockWasmBudget struct {
	mu        sync.Mutex
	on        bool
	height    int64
	remaining uint64
}

// reset starts a block: on reports the switch, limit is the budget in VM gas.
func (b *blockWasmBudget) reset(height int64, on bool, limit uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.on, b.height, b.remaining = on, height, limit
}

func (b *blockWasmBudget) counts(ctx sdk.Context) bool {
	return b.on && ctx.ExecMode() == sdk.ExecModeFinalize && len(ctx.TxBytes()) > 0 && ctx.BlockHeight() == b.height
}

// Limit gives a VM call at most what is left of the block's budget.
func (b *blockWasmBudget) Limit(ctx sdk.Context, gasLimit uint64) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return gasLimit, nil
	}
	if b.remaining == 0 {
		return 0, ErrBlockWasmBudget.Wrap("this block's CosmWasm execution budget is spent; retry in the next block")
	}
	if gasLimit > b.remaining {
		return b.remaining, nil
	}
	return gasLimit, nil
}

// Used takes a VM call's gas from the budget.
func (b *blockWasmBudget) Used(ctx sdk.Context, gasUsed uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return
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

// wasmBlockBudget returns the app's one budget.
func (app *QoreChainApp) wasmBlockBudget() *blockWasmBudget {
	if app.blockBudget == nil {
		app.blockBudget = &blockWasmBudget{}
	}
	return app.blockBudget
}

// resetWasmBlockBudget starts the block's budget from ctx's state. It runs
// after the gas multiplier is set, so the budget converts to VM gas at the
// block's price. Reads are free of gas.
func (app *QoreChainApp) resetWasmBlockBudget(ctx sdk.Context) {
	on := app.PQCKeeper != nil && app.CrossVMKeeper != nil &&
		app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureWasmBlockGasBudget)
	var limit uint64
	if on {
		limit = app.wasmGasRegister().ToWasmVMGas(app.CrossVMKeeper.GetParams(ctx).BlockWasmGasLimit())
	}
	app.wasmBlockBudget().reset(ctx.BlockHeight(), on, limit)
}
