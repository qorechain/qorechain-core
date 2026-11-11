package app

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"math"
	"sync"
	"time"

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
	uploadPool uint64 // compiling uploads; not counted for congestion
	uploadUsed bool   // an upload has been charged to the pool this block
	txCap      uint64
	blockLevel uint64
	tx         wasmTxKey
	txUsed     uint64
	nextToken  uint64
	uploadMult uint64 // VM gas per budget unit (the base multiplier)
	// payers is the VM gas each fee payer's transactions used in the block,
	// for congestion pricing (who exhausted the pool).
	payers map[string]uint64
	reg    *calibratedGasRegister
	// simMargin is whether simulations add the congestion margin
	// (wasm-congestion-pricing on), set with the multiplier.
	simMargin bool
	// queryLoads bounds code loading by queries (node-local).
	queryLoads loadAllowance
	// recent is the codes this node loaded lately (node-local), bounded by
	// its memory cache size: queries of them load nothing.
	recent recentLoads
	// loaded is the set of codes whose load the block has charged, in
	// block execution order (reset with the block), with the number of
	// repeated calls each has had since; loadedBytes their estimated
	// compiled size together.
	loaded      map[string]uint32
	loadedBytes uint64
}

type wasmTxPayerKey struct{}

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
func (b *blockWasmBudget) reset(height int64, on bool, total uint64, reserveBps, shareBps uint32, blockLevel, unit uint64, uploadBps uint32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.uploadMult = unit
	b.on, b.height, b.total, b.blockLevel = on, height, total, blockLevel
	b.queuePool = bpsOf(total, reserveBps)
	b.txPool = total - b.queuePool
	b.uploadPool, b.uploadUsed = bpsOf(b.txPool, uploadBps), false
	b.txCap = bpsOf(total, shareBps)
	b.tx, b.txUsed, b.nextToken = wasmTxKey{}, 0, 0
	b.payers = map[string]uint64{}
	b.loaded, b.loadedBytes = map[string]uint32{}, 0
}

func (b *blockWasmBudget) counts(ctx sdk.Context) bool {
	return b.on && ctx.ExecMode() == sdk.ExecModeFinalize && ctx.BlockHeight() == b.height
}

// withTxToken gives a transaction executing in the block its identity for
// the per-transaction share, once, in the ante handler: the block's
// transactions are numbered in order. Nothing is hashed per VM call.
func (b *blockWasmBudget) withTxToken(ctx sdk.Context, tx sdk.Tx) sdk.Context {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return ctx
	}
	b.nextToken++
	ctx = ctx.WithValue(wasmTxTokenKey{}, b.nextToken)
	if feeTx, ok := tx.(sdk.FeeTx); ok {
		ctx = ctx.WithValue(wasmTxPayerKey{}, string(feeTx.FeePayer()))
	}
	return ctx
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

// Used takes a VM call's internal gas from its pool (external gas is the
// storage and query work the SDK meter already prices; nested calls count
// themselves). A transaction's call that was cut short by the budget and
// used all it was given, inside and outside the VM, aborts the transaction.
func (b *blockWasmBudget) Used(ctx sdk.Context, gasUsed, gasExternal, limit uint64, clamped bool) {
	b.mu.Lock()
	if !b.counts(ctx) {
		margin, reg := b.simMargin, b.reg
		b.mu.Unlock()
		// A simulation is priced two exhaustion steps (x2.25) above the
		// factor in force (wasm-congestion-pricing): the factor can rise by
		// half per block before the transaction lands, and a tightly
		// estimated contract call would otherwise run out of gas.
		if margin && reg != nil && ctx.ExecMode() == sdk.ExecModeSimulate {
			if m := reg.Multiplier(); m > 0 {
				meter := ctx.GasMeter()
				meter.ConsumeGas(min64(gasUsed/m*5/4, meter.GasRemaining()), "wasm simulation congestion margin")
			}
		}
		return
	}
	abort := false
	switch {
	case len(ctx.TxBytes()) > 0:
		if txKeyOf(ctx) == b.tx {
			b.txUsed += min64(gasUsed, math.MaxUint64-b.txUsed)
		}
		if payer, ok := ctx.Value(wasmTxPayerKey{}).(string); ok && b.payers != nil {
			b.payers[payer] += min64(gasUsed, math.MaxUint64-b.payers[payer])
		}
		b.txPool -= min64(gasUsed, b.txPool)
		// The call ran out of what the budget gave it when its internal and
		// external gas together reached the limit: wasmvm reports only the
		// internal part, so a contract that touched storage before running
		// out reports less than the limit.
		total := gasUsed + min64(gasExternal, math.MaxUint64-gasUsed)
		abort = clamped && total >= limit
	case crossvmtypes.IsQueueExecution(ctx):
		fromQueue := min64(gasUsed, b.queuePool)
		b.queuePool -= fromQueue
		b.txPool -= min64(gasUsed-fromQueue, b.txPool)
	default:
		b.blockLevel -= min64(gasUsed, b.blockLevel)
	}
	reg := b.reg
	b.mu.Unlock()
	if abort {
		// The aborted call's VM work goes on the transaction's meter (wasmd
		// would have charged it after the call returned), so the gas the
		// transaction reports used includes it.
		if reg != nil {
			if m := reg.Multiplier(); m > 0 {
				meter := ctx.GasMeter()
				charge := min64(gasUsed/m, meter.GasRemaining())
				meter.ConsumeGas(charge, "wasm call cut by the block budget")
			}
		}
		abortTx(fmt.Sprintf("a contract call was cut at %d VM gas by the block's CosmWasm execution budget", limit))
	}
}

// UploadBudgetUnitsPerByte is what one upload unit (a byte of module, or
// wasmfilter.CompiledBytesPerUploadUnit bytes of its conservative compiled
// size, whichever is more) takes from the block's upload pool, in budget
// units (SDK gas of VM work at the base price). Measured compile cost is
// well under the 265 ns design bound per unit: a maximum-size module
// (800 KiB) takes 819,200 units, and the largest upload within the caps
// 1,048,576, of the default pool of 1,125,000.
const UploadBudgetUnitsPerByte = 1

// ChargeUpload takes compiling a module of the given upload units
// (wasmfilter: its bytes, or its compiled size over
// CompiledBytesPerUploadUnit) from the block's upload pool (a call outside a
// transaction from the block-level counter). The upload pool is separate
// from the execution pools and does not count towards congestion, so
// uploads cannot close the execution lane or set the price; it bounds the
// compile CPU of a block. The first upload of a block may take up to
// wasmfilter.MaxUploadUnits even past the pool (it then empties the pool),
// so an upload within the caps is never too large for every block. A
// transaction's upload that does not fit aborts the transaction; any other
// is refused.
func (b *blockWasmBudget) ChargeUpload(ctx sdk.Context, units int) error {
	return b.upload(ctx, units, true)
}

// UploadFits is ChargeUpload without taking anything: the same refusal (or
// abort) when the upload would not fit now, nil otherwise.
func (b *blockWasmBudget) UploadFits(ctx sdk.Context, units int) error {
	return b.upload(ctx, units, false)
}

func (b *blockWasmBudget) upload(ctx sdk.Context, units int, take bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) || units <= 0 {
		return nil
	}
	need := saturatingVMGas(uint64(units)*UploadBudgetUnitsPerByte, b.uploadMult)
	switch {
	case len(ctx.TxBytes()) > 0, crossvmtypes.IsQueueExecution(ctx):
		fits := need <= b.uploadPool
		if !fits && !b.uploadUsed && need <= saturatingVMGas(wasmfilter.MaxUploadUnits, b.uploadMult) {
			fits = true // the block's first upload, within the caps
		}
		if !fits {
			if len(ctx.TxBytes()) > 0 {
				abortTx("storing this code does not fit in what is left of the block's code-upload budget; retry in a later block")
			}
			return ErrBlockWasmBudget.Wrap("storing this code does not fit in what is left of the block's code-upload budget")
		}
		if take {
			b.uploadPool -= min64(need, b.uploadPool)
			b.uploadUsed = true
		}
	default:
		if need > b.blockLevel {
			return ErrBlockWasmBudget.Wrap("storing this code does not fit in this block's CosmWasm budget for calls outside transactions")
		}
		if take {
			b.blockLevel -= need
		}
	}
	return nil
}

// UploadRemaining is what is left of the block's upload pool, in VM gas.
func (b *blockWasmBudget) UploadRemaining() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.uploadPool
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
// was used, and whether the transactions' pool was exhausted (less than 5
// percent left) by at least two fee payers each using at least
// CongestionPayerShareBps of the budget, for congestion pricing. One payer
// exhausting the pool alone counts as ordinary high use.
func (b *blockWasmBudget) utilisation(ctx sdk.Context) (used, total uint64, exhaustedByMany bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.counts(ctx) {
		return 0, 0, false
	}
	used, total = b.total-(b.txPool+b.queuePool), b.total
	if total == 0 || b.txPool >= total/20 {
		return used, total, false
	}
	material := 0
	for _, u := range b.payers {
		if bpsOf(total, uint32(crossvmtypes.CongestionPayerShareBps)) <= u {
			material++
		}
	}
	return used, total, material >= 2
}

// wasmBlockBudget returns the app's one budget.
func (app *QoreChainApp) wasmBlockBudget() *blockWasmBudget {
	if app.blockBudget == nil {
		app.blockBudget = &blockWasmBudget{reg: app.wasmGasRegister()}
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
	var reserve, share, uploadBps uint32
	if on {
		p := app.CrossVMKeeper.GetParams(ctx)
		m := app.wasmGasRegister().BaseMultiplier()
		total = saturatingVMGas(p.BlockWasmGasLimit(), m)
		blockLevel = saturatingVMGas(p.BlockLevelWasmGasLimit(), m)
		reserve, share = p.QueueWasmReserve(), p.TxWasmShareBps()
		uploadBps = p.BlockUploadBps()
	}
	app.wasmBlockBudget().reset(ctx.BlockHeight(), on, total, reserve, share, blockLevel, app.wasmGasRegister().BaseMultiplier(), uploadBps)
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

// queueWasmMeter is, for a queued call into CosmWasm asking for units of
// contract execution (base gas units), the SDK gas meter that buys exactly
// that at the price in force, or zero when the block's wasm budget cannot
// admit all of it (the call then waits for a later block).
func (app *QoreChainApp) queueWasmMeter(ctx sdk.Context, units uint64) uint64 {
	reg := app.wasmGasRegister()
	base, eff := reg.BaseMultiplier(), reg.Multiplier()
	vm := saturatingVMGas(units, base)
	if app.wasmBlockBudget().admit(ctx, vm) < vm || eff == 0 {
		return 0
	}
	return vm/eff + min64(vm%eff, 1)
}

// wasmTxToken is the ante step that numbers the block's transactions for the
// per-transaction share.
func (app *QoreChainApp) wasmTxToken(ctx sdk.Context, tx sdk.Tx) sdk.Context {
	return app.wasmBlockBudget().withTxToken(ctx, tx)
}

// budgetAbortRecovery turns a wasmfilter.BudgetAbort raised during a
// transaction into the transaction's ordinary failure (run-tx recovery).
func budgetAbortRecovery(r any) error {
	if a, ok := r.(wasmfilter.BudgetAbort); ok {
		return ErrBlockWasmBudget.Wrap(a.Reason)
	}
	return nil
}

// setSimulationMargin records whether simulations add the congestion margin.
func (b *blockWasmBudget) setSimulationMargin(on bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.simMargin = on
}

// QueryLoadUnitsPerSecond is how much code loading (wasmfilter.LoadUnits of
// unpinned codes) smart queries served by this node may cause per second,
// all queries together. Node-local: a query that fans out to many unpinned
// codes cannot make a public RPC node load modules without bound; it fails
// and can be retried.
const QueryLoadUnitsPerSecond = 1_000_000

// ChargeSetup charges loading an unpinned code before a call
// (wasm-code-load-priced): units from the block's budget like the call's own
// execution (the transaction's share and pool, the queue's, or the
// block-level counter, refused or aborting as Limit does), and gas at the
// price in force on ctx's meter. The gas is taken first: a call that cannot
// pay it fails before the budget or the set of loaded codes is touched.
//
// In block execution a code's load is charged in full on its first call in
// the block. While the block's distinct loads stay within what a node's
// memory cache holds (FreeRepeatLoadBytes), its next FreeRepeatLoadsPerBlock
// calls pay nothing and every later one pays RepeatLoadFraction of the
// load: queries and simulations served beside the block share the node's
// cache and can evict a code between two calls, so a repeat may load again,
// and the fraction bounds that. Past FreeRepeatLoadBytes every call is
// charged in full. The count is the block's own (block execution order,
// reset every block), never the node's cache state. Simulation and CheckTx
// always charge, as for a first call, and mark nothing. A query served by
// the node pays only for a code that is not hot (loaded in this block or
// recently on this node), and such loads share a node-local allowance per
// second.
func (b *blockWasmBudget) ChargeSetup(ctx sdk.Context, checksum []byte, units, gas, compiled uint64) error {
	key := string(checksum)
	b.mu.Lock()
	counts := b.counts(ctx)
	query := !counts && ctx.IsCheckTx() && len(ctx.TxBytes()) == 0 && ctx.ExecMode() == sdk.ExecModeCheck
	repeats, seen := b.loaded[key]
	if counts && seen && b.loadedBytes <= FreeRepeatLoadBytes {
		repeats++
		b.loaded[key] = repeats
		b.recent.touch(key, compiled)
		if repeats <= FreeRepeatLoadsPerBlock {
			b.mu.Unlock()
			return nil
		}
		units, gas = repeatLoadShare(units), repeatLoadShare(gas)
	}
	if query && (seen || b.recent.has(key)) {
		b.recent.touch(key, compiled)
		b.mu.Unlock()
		return nil
	}
	base, eff := CalibratedWasmGasMultiplier, CalibratedWasmGasMultiplier
	if b.reg != nil {
		base, eff = b.reg.BaseMultiplier(), b.reg.Multiplier()
	}
	b.mu.Unlock()

	// The gas at the price in force (gas x base / effective, the congestion
	// factor), before anything else.
	sdkGas := gas
	if eff > 0 {
		g := saturatingVMGas(gas, base)
		sdkGas = g/eff + min64(g%eff, 1)
	}
	if !counts && !query {
		ctx.GasMeter().ConsumeGas(sdkGas, "wasm code load")
		// A simulation or CheckTx loads the code on this node too.
		b.mu.Lock()
		b.recent.touch(key, compiled)
		b.mu.Unlock()
		return nil
	}
	if query {
		b.mu.Lock()
		ok := b.queryLoads.take(units)
		b.mu.Unlock()
		if !ok {
			return ErrBlockWasmBudget.Wrap("this node's allowance for loading contract code in queries is spent; retry shortly")
		}
		ctx.GasMeter().ConsumeGas(sdkGas, "wasm code load")
		b.mu.Lock()
		b.recent.touch(key, compiled)
		b.mu.Unlock()
		return nil
	}
	ctx.GasMeter().ConsumeGas(sdkGas, "wasm code load")
	return b.takeSetup(ctx, key, saturatingVMGas(units, base), compiled)
}

// FreeRepeatLoadsPerBlock is how many calls of a code after its first in a
// block load nothing; RepeatLoadFraction is the share of the load every
// later call in the block pays (wasm-code-load-priced).
const (
	FreeRepeatLoadsPerBlock = 8
	RepeatLoadFraction      = 8 // one eighth
)

// repeatLoadShare is a repeated call's share of a load cost, at least 1.
func repeatLoadShare(v uint64) uint64 {
	return max(v/RepeatLoadFraction, min64(v, 1))
}

// takeSetup takes a charged load's VM gas from the block's budget and marks
// the code loaded for the rest of the block.
func (b *blockWasmBudget) takeSetup(ctx sdk.Context, key string, vm, compiled uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.takeUnitsLocked(ctx, vm, "loading this contract's code"); err != nil {
		return err
	}
	// Charged: the load happens (whether the transaction then succeeds or
	// not), so the code counts as loaded for the rest of the block.
	if _, seen := b.loaded[key]; !seen && b.loaded != nil {
		b.loaded[key] = 0
		b.loadedBytes += min64(compiled, math.MaxUint64-b.loadedBytes)
	}
	b.recent.touch(key, compiled)
	return nil
}

// takeUnitsLocked takes vm (VM gas) from the pool of ctx's kind of call: a
// transaction's share and pool (aborting the transaction when it does not
// fit), the queue's reserve and then the transactions' pool, or the
// block-level counter. what names the work in the refusal.
func (b *blockWasmBudget) takeUnitsLocked(ctx sdk.Context, vm uint64, what string) error {
	switch {
	case len(ctx.TxBytes()) > 0:
		if k := txKeyOf(ctx); k != b.tx {
			b.tx, b.txUsed = k, 0
		}
		if vm > b.txPool {
			abortTx(what + " does not fit in what is left of the block's CosmWasm execution budget; retry in a later block")
		}
		if b.txUsed+vm > b.txCap {
			abortTx("this transaction has used its share of the block's CosmWasm execution budget")
		}
		b.txUsed += vm
		if payer, ok := ctx.Value(wasmTxPayerKey{}).(string); ok && b.payers != nil {
			b.payers[payer] += vm
		}
		b.txPool -= vm
	case crossvmtypes.IsQueueExecution(ctx):
		if vm > b.queuePool+b.txPool {
			return ErrBlockWasmBudget.Wrap(what + " does not fit in what is left of the block's CosmWasm execution budget")
		}
		fromQueue := min64(vm, b.queuePool)
		b.queuePool -= fromQueue
		b.txPool -= vm - fromQueue
	default:
		if vm > b.blockLevel {
			return ErrBlockWasmBudget.Wrap(what + " does not fit in this block's CosmWasm budget for calls outside transactions")
		}
		b.blockLevel -= vm
	}
	return nil
}

// ChargeCallSetup charges setting up one call (wasm-call-setup-priced):
// gas at the price in force on ctx's meter first (in every mode: block
// execution, simulation, CheckTx and queries), then, in block execution
// only, units from the call's pool as takeUnitsLocked does. Every call is
// charged, repeated calls and pinned codes included: each runs in a fresh
// instance.
func (b *blockWasmBudget) ChargeCallSetup(ctx sdk.Context, units, gas uint64) error {
	b.mu.Lock()
	counts := b.counts(ctx)
	base, eff := CalibratedWasmGasMultiplier, CalibratedWasmGasMultiplier
	if b.reg != nil {
		base, eff = b.reg.BaseMultiplier(), b.reg.Multiplier()
	}
	b.mu.Unlock()
	sdkGas := gas
	if eff > 0 {
		g := saturatingVMGas(gas, base)
		sdkGas = g/eff + min64(g%eff, 1)
	}
	ctx.GasMeter().ConsumeGas(sdkGas, "wasm call setup")
	if !counts {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.takeUnitsLocked(ctx, saturatingVMGas(units, base), "setting up this contract call")
}

// FreeRepeatLoadBytes is the conservative compiled size of the distinct
// codes a block may load before repeated calls to them are charged again:
// an eighth of the minimum memory cache (MinWasmMemoryCacheMiB), so a code
// loaded earlier in the block is still in memory when called again even if
// real compiled sizes were well above the estimate.
const FreeRepeatLoadBytes = 32 << 20

// loadAllowance is a per-second allowance (node-local, wall clock).
type loadAllowance struct {
	second int64
	used   uint64
}

func (a *loadAllowance) take(units uint64) bool {
	now := time.Now().Unix()
	if now != a.second {
		a.second, a.used = now, 0
	}
	if a.used+units > QueryLoadUnitsPerSecond {
		return false
	}
	a.used += units
	return true
}

// recentLoads is a node-local least-recently-used set of loaded codes, by
// conservative compiled size, bounded by the node's wasm memory cache size.
type recentLoads struct {
	capacity uint64
	total    uint64
	order    *list.List
	byKey    map[string]*list.Element
}

type recentLoad struct {
	key  string
	size uint64
}

func (r *recentLoads) has(key string) bool {
	_, ok := r.byKey[key]
	return ok
}

func (r *recentLoads) touch(key string, size uint64) {
	if r.order == nil {
		r.order, r.byKey = list.New(), map[string]*list.Element{}
	}
	if r.capacity == 0 {
		r.capacity = uint64(MinWasmMemoryCacheMiB) << 19
	}
	if e, ok := r.byKey[key]; ok {
		r.order.MoveToFront(e)
		return
	}
	r.byKey[key] = r.order.PushFront(recentLoad{key, size})
	r.total += size
	for r.total > r.capacity && r.order.Len() > 0 {
		last := r.order.Back()
		v := last.Value.(recentLoad)
		r.order.Remove(last)
		delete(r.byKey, v.key)
		r.total -= v.size
	}
}

// setRecentLoadCapacity sizes the recent-loads set from the node's memory
// cache size in bytes: half of it, so a code the set calls hot is still in
// memory even if real compiled sizes were twice the estimate.
func (b *blockWasmBudget) setRecentLoadCapacity(bytes uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recent.capacity = bytes / 2
}

// resetNodeLocalLoads empties the node-local load state (the recent-loads
// set and the query allowance). For tests that share one app.
func (b *blockWasmBudget) resetNodeLocalLoads() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recent = recentLoads{capacity: b.recent.capacity}
	b.queryLoads = loadAllowance{}
}
