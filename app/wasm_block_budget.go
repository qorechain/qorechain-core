package app

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sync"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	crossvmtypes "github.com/qorechain/qorechain-core/x/crossvm/types"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	"github.com/qorechain/qorechain-core/x/wasmfilter"
)

// ErrBlockWasmBudget is the failure of a CosmWasm call, or of the whole
// transaction, once the block's wasm execution budget, or the calling
// transaction's share of it, is spent (wasm-block-gas-budget).
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
// The budget is split in two pools, and calls outside a transaction have a
// third counter of their own:
//
//   - transactions draw from the transaction pool (the budget less the
//     queue's reserve, queue_wasm_reserve_bps), one transaction at most its
//     share of the whole budget (max_tx_wasm_share_bps). When a
//     transaction's call is refused, or cut short and runs out, the whole
//     transaction is aborted (wasmfilter.BudgetAbort), so nothing it
//     started (a submessage, an IBC packet receive) can turn the cut into a
//     handled error: the transaction fails and can be retried in a later
//     block;
//   - the end-of-block async queue draws from its reserve and then from what
//     transactions left; it admits a queued call on at most that much
//     (queueWasmAdmit), so a call is never starved out by transactions;
//   - any other call outside a transaction (a governance proposal executed
//     at the end of the block, with every submessage and reply it makes)
//     draws from its own per-block counter (max_block_level_wasm_gas) and is
//     refused when it is spent; transactions cannot spend it, so they cannot
//     make a proposal fail, and a proposal cannot run the block without
//     bound.
type blockWasmBudget struct {
	mu         sync.Mutex
	on         bool
	height     int64
	total      uint64 // transaction pool + queue reserve at the start
	txPool     uint64
	queuePool  uint64
	txCap      uint64
	blockLevel uint64
	tx         wasmTxKey
	txUsed     uint64
	nextToken  uint64
}

// wasmTxKey identifies the transaction a call belongs to.
type wasmTxKey struct {
	token uint64
	hash  [32]byte
}

type wasmTxTokenKey struct{}

// saturatingVMGas converts SDK gas to VM gas at multiplier m without
// overflowing.
func saturatingVMGas(sdkGas, m uint64) uint64 {
	if m != 0 && sdkGas > math.MaxUint64/m {
		return math.MaxUint64
	}
	return sdkGas * m
}

func min64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func bpsOf(v uint64, bps uint32) uint64 {
	if v > math.MaxUint64/10_000 {
		return v / 10_000 * uint64(bps)
	}
	return v * uint64(bps) / 10_000
}

// reset starts a block. Limits are in VM gas.
func (b *blockWasmBudget) reset(height int64, on bool, total uint64, reserveBps, shareBps uint32, blockLevel uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.on, b.height, b.total, b.blockLevel = on, height, total, blockLevel
	b.queuePool = bpsOf(total, reserveBps)
	b.txPool = total - b.queuePool
	b.txCap = bpsOf(total, shareBps)
	b.tx, b.txUsed, b.nextToken = wasmTxKey{}, 0, 0
}

func (b *blockWasmBudget) counts(ctx sdk.Context) bool {
	return b.on && ctx.ExecMode() == sdk.ExecModeFinalize && ctx.BlockHeight() == b.height
}

// withTxToken gives a transaction executing in the block its identity for
// the per-transaction share, once, in the ante handler: the block's
// transactions are numbered in order. Nothing is hashed per VM call.
func (b *blockWasmBudget) withTxToken(ctx sdk.Context) sdk.Context {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return ctx
	}
	b.nextToken++
	return ctx.WithValue(wasmTxTokenKey{}, b.nextToken)
}

// txKeyOf is the transaction's identity: its token, or (for a context that
// did not pass the ante handler, which block execution does not produce)
// the hash of its bytes.
func txKeyOf(ctx sdk.Context) wasmTxKey {
	if t, ok := ctx.Value(wasmTxTokenKey{}).(uint64); ok && t != 0 {
		return wasmTxKey{token: t}
	}
	return wasmTxKey{hash: sha256.Sum256(ctx.TxBytes())}
}

func abortTx(reason string) {
	panic(wasmfilter.BudgetAbort{Reason: reason})
}

// Limit gives a VM call at most what its kind may still use.
func (b *blockWasmBudget) Limit(ctx sdk.Context, gasLimit uint64) (uint64, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return gasLimit, false, nil
	}
	var limit uint64
	switch {
	case len(ctx.TxBytes()) > 0:
		if key := txKeyOf(ctx); key != b.tx {
			b.tx, b.txUsed = key, 0
		}
		if b.txPool == 0 {
			abortTx("this block's CosmWasm execution budget is spent; retry in a later block")
		}
		if b.txUsed >= b.txCap {
			abortTx("this transaction has used its share of the block's CosmWasm execution budget")
		}
		limit = min64(gasLimit, min64(b.txPool, b.txCap-b.txUsed))
	case crossvmtypes.IsQueueExecution(ctx):
		avail := b.queuePool + b.txPool
		if avail == 0 {
			return 0, false, ErrBlockWasmBudget.Wrap("this block's CosmWasm execution budget is spent")
		}
		limit = min64(gasLimit, avail)
	default:
		if b.blockLevel == 0 {
			return 0, false, ErrBlockWasmBudget.Wrap("this block's CosmWasm budget for calls outside transactions is spent")
		}
		limit = min64(gasLimit, b.blockLevel)
	}
	return limit, limit < gasLimit, nil
}

// Used takes a VM call's gas from its pool. A transaction's call that was
// cut short by the budget and used all it was given aborts the transaction.
func (b *blockWasmBudget) Used(ctx sdk.Context, gasUsed, limit uint64, clamped bool) {
	b.mu.Lock()
	if !b.counts(ctx) {
		b.mu.Unlock()
		return
	}
	abort := false
	switch {
	case len(ctx.TxBytes()) > 0:
		if txKeyOf(ctx) == b.tx {
			b.txUsed += min64(gasUsed, math.MaxUint64-b.txUsed)
		}
		b.txPool -= min64(gasUsed, b.txPool)
		abort = clamped && gasUsed >= limit
	case crossvmtypes.IsQueueExecution(ctx):
		fromQueue := min64(gasUsed, b.queuePool)
		b.queuePool -= fromQueue
		b.txPool -= min64(gasUsed-fromQueue, b.txPool)
	default:
		b.blockLevel -= min64(gasUsed, b.blockLevel)
	}
	b.mu.Unlock()
	if abort {
		abortTx(fmt.Sprintf("a contract call was cut at %d VM gas by the block's CosmWasm execution budget", limit))
	}
}

// Remaining is what is left for transactions and the queue together, in VM
// gas. For tests and tools.
func (b *blockWasmBudget) Remaining() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.txPool + b.queuePool
}

// BlockLevelRemaining is what is left for calls outside transactions.
func (b *blockWasmBudget) BlockLevelRemaining() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.blockLevel
}

// admit is how much VM gas a queued call may use: at most vmGas, from the
// queue's reserve and then what transactions left (all of vmGas when the
// budget is off or ctx is not block execution).
func (b *blockWasmBudget) admit(ctx sdk.Context, vmGas uint64) uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return vmGas
	}
	return min64(vmGas, b.queuePool+b.txPool)
}

// utilisation is how much of this block's budget (transactions and queue)
// was used, for congestion pricing.
func (b *blockWasmBudget) utilisation(ctx sdk.Context) (used, total uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return 0, 0
	}
	return b.total - (b.txPool + b.queuePool), b.total
}

// wasmBlockBudget returns the app's one budget.
func (app *QoreChainApp) wasmBlockBudget() *blockWasmBudget {
	if app.blockBudget == nil {
		app.blockBudget = &blockWasmBudget{}
	}
	return app.blockBudget
}

// resetWasmBlockBudget starts the block's budget from ctx's state. It runs
// after the gas multiplier is set and converts at the base multiplier (before
// congestion pricing), saturating rather than overflowing. Reads are free of
// gas.
func (app *QoreChainApp) resetWasmBlockBudget(ctx sdk.Context) {
	on := app.PQCKeeper != nil && app.CrossVMKeeper != nil &&
		app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureWasmBlockGasBudget)
	var total, blockLevel uint64
	var reserve, share uint32
	if on {
		p := app.CrossVMKeeper.GetParams(ctx)
		m := app.wasmGasRegister().BaseMultiplier()
		total = saturatingVMGas(p.BlockWasmGasLimit(), m)
		blockLevel = saturatingVMGas(p.BlockLevelWasmGasLimit(), m)
		reserve, share = p.QueueWasmReserve(), p.TxWasmShareBps()
	}
	app.wasmBlockBudget().reset(ctx.BlockHeight(), on, total, reserve, share, blockLevel)
}

// queueWasmAdmit is how much SDK gas a queued call (prepaid sdkGas) may run
// on this block under the wasm budget, at the multiplier in force; zero
// leaves it pending.
func (app *QoreChainApp) queueWasmAdmit(ctx sdk.Context, sdkGas uint64) uint64 {
	m := app.wasmGasRegister().Multiplier()
	vm := app.wasmBlockBudget().admit(ctx, saturatingVMGas(sdkGas, m))
	if m == 0 {
		return sdkGas
	}
	admitted := min64(sdkGas, vm/m)
	if admitted < min64(sdkGas, crossvmtypes.MinQueueAdmission) {
		return 0
	}
	return admitted
}

// wasmTxToken is the ante step that numbers the block's transactions for the
// per-transaction share.
func (app *QoreChainApp) wasmTxToken(ctx sdk.Context) sdk.Context {
	return app.wasmBlockBudget().withTxToken(ctx)
}

// budgetAbortRecovery turns a wasmfilter.BudgetAbort raised during a
// transaction into the transaction's ordinary failure (run-tx recovery).
func budgetAbortRecovery(r any) error {
	if a, ok := r.(wasmfilter.BudgetAbort); ok {
		return ErrBlockWasmBudget.Wrap(a.Reason)
	}
	return nil
}
