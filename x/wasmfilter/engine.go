package wasmfilter

import (
	"fmt"
	"math"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
)

// Filter holds the switch and the per-checksum verdicts shared by the engine
// decorator and the message server decorator.
type Filter struct {
	uploadBounds    func(sdk.Context) bool
	uploadPermitted func(ctx sdk.Context, sender string) bool
	on              func(sdk.Context) bool
	noFloat         func(sdk.Context) bool
	budget          Budget
	deserScale      func() (num, den uint64)
	verdicts        *Verdicts
	loadPriced      func(sdk.Context) bool
	pinned          func(ctx sdk.Context, checksum []byte) bool
	compiledSize    func(sdk.Context) bool
	shape           func(sdk.Context) bool
	setupPriced     func(sdk.Context) bool
}

// CallSetupCharger is a Budget that also charges setting up a call
// (wasm-call-setup-priced): units from the block's budget like the call's
// own execution (refused or aborting as Limit does) and gas (SDK gas at the
// base price, scaled by the price in force) from ctx's meter, the gas first.
// Outside block execution only the gas is charged.
type CallSetupCharger interface {
	ChargeCallSetup(ctx sdk.Context, units, gas uint64) error
}

// WithCallSetupPricing charges every call its setup (CallSetupUnits, and
// CallSetupGasPerUnit gas per unit) while on(ctx) is true, pinned codes
// included: every call gets a fresh instance whatever cache it came from.
func (f *Filter) WithCallSetupPricing(on func(sdk.Context) bool) *Filter {
	f.setupPriced = on
	return f
}

// CallSetupCost is what setting up one call of a code costs (budget units
// and gas at the base price), from the filter's verdict for its checksum;
// code reads the code's bytes when the verdict is not cached. False when
// the code cannot be read.
func (f *Filter) CallSetupCost(checksum []byte, code func() ([]byte, error)) (units, gas uint64, ok bool) {
	known, v := f.verdictFor(checksum, code)
	if !known {
		return 0, 0, false
	}
	return v.SetupUnits, v.SetupUnits * CallSetupGasPerUnit, true
}

// CallGrowthCharger is a CallSetupCharger that also charges, after a call,
// the memory it grew into (wasm-call-setup-priced): gas first (out of gas
// when the meter cannot cover it), then, in block execution, units from the
// call's pool; when they do not fit, what is left of that pool (or the
// transaction's share) is taken and the transaction aborted, or the call
// refused outside a transaction.
type CallGrowthCharger interface {
	ChargeCallGrowth(ctx sdk.Context, units, gas uint64) error
}

// callSetup is one call's setup charge in progress: the querier handed to
// the VM (which reports the call's memory size) and what is needed to
// charge the growth after the call.
type callSetup struct {
	ctx     sdk.Context
	initial uint64
	q       *pagesQuerier
}

// pagesQuerier is the querier handed to the VM while call setup is priced:
// the querier wasmd made, which also takes the call's memory size from the
// patched VM (wasmvmtypes.MemoryPagesReporter).
type pagesQuerier struct {
	wasmvm.Querier
	pages uint64
}

func (q *pagesQuerier) ReportMemoryPages(pages uint64) { q.pages = pages }

// chargeCallSetup charges setting up one call, before it runs, and returns
// the querier to run it with and the state for chargeCallGrowth (nil when
// call setup is not priced in this context).
func (e *Engine) chargeCallSetup(checksum wasmvm.Checksum, querier wasmvm.Querier) (wasmvm.Querier, *callSetup, error) {
	if e.f.setupPriced == nil {
		return querier, nil, nil
	}
	sc, ok := e.f.budget.(CallSetupCharger)
	if !ok {
		return querier, nil, nil
	}
	ctx, ok := contextOf(querier)
	if !ok || !e.f.setupPriced(ctx) {
		return querier, nil, nil
	}
	known, v := e.verdict(checksum)
	if !known {
		return querier, nil, nil
	}
	if err := sc.ChargeCallSetup(ctx, v.SetupUnits, v.SetupUnits*CallSetupGasPerUnit); err != nil {
		return querier, nil, err
	}
	pq := &pagesQuerier{Querier: querier}
	return pq, &callSetup{ctx: ctx, initial: v.MemoryPages, q: pq}, nil
}

// afterCall settles one call with the budget, in an order that lets nothing
// skip a charge for work already done. First the call's own VM gas is
// recorded (budgetUsed); an abort it raises (a call cut short by the
// budget) is held. Then the memory the call grew into is charged
// (chargeCallGrowth: its units, then its gas). Then the held abort is
// raised, also when the growth charge panicked (out of gas), so a
// submessage that recovers an out-of-gas cannot swallow it. ext is the
// call's external gas, read before the growth charge adds to the meter.
func (e *Engine) afterCall(querier wasmvm.Querier, cs *callSetup, used, ext, limit uint64, clamped bool, err error) error {
	var held any
	func() {
		defer func() {
			if r := recover(); r != nil {
				if !IsBudgetAbort(r) {
					panic(r)
				}
				held = r
			}
		}()
		e.budgetUsed(querier, used, ext, limit, clamped)
	}()
	if held != nil {
		defer panic(held)
	}
	return e.chargeCallGrowth(cs, err)
}

// chargeCallGrowth charges, after a call, the memory it grew into beyond
// its initial memory (CallGrowthUnits), whether the call succeeded or not;
// err is the call's own error, returned unless the charge fails.
func (e *Engine) chargeCallGrowth(cs *callSetup, err error) error {
	if cs == nil {
		return err
	}
	units := CallGrowthUnits(cs.initial, cs.q.pages)
	if units == 0 {
		return err
	}
	gc, ok := e.f.budget.(CallGrowthCharger)
	if !ok {
		return err
	}
	if cerr := gc.ChargeCallGrowth(cs.ctx, units, units*CallSetupGasPerUnit); cerr != nil {
		return cerr
	}
	return err
}

// WithCompiledSizeCharge turns on, while on(ctx) is true and upload bounds
// are on, the function cap and the compiled-size charge of uploads
// (wasm-upload-compiled-size).
func (f *Filter) WithCompiledSizeCharge(on func(sdk.Context) bool) *Filter {
	f.compiledSize = on
	return f
}

// Loading a code that is not pinned (from the node's disk cache when it is
// not in memory) is work proportional to the compiled module
// (ConservativeCompiledSize); wasmd prices it flat. From
// wasm-code-load-priced every call to an unpinned code takes
// LoadUnits(compiled) from the block wasm budget and LoadGasPerUnit gas per
// unit at the price in force, before the call. Calibrated on measured loads
// (2.7 to 8.8 compiled KB per microsecond on the measuring host, validator
// hardware about three times slower) against the budget's design bound of
// 265 ns per unit: about one unit per LoadCompiledBytesPerUnit compiled
// bytes. The gas makes a cold load cost about as much per nanosecond as
// executing code (within 2x on the measured shapes).
const (
	LoadUnitsBase            = 1_000
	LoadCompiledBytesPerUnit = 550 // 450 real compiled bytes: real is at most 0.81 of the estimate
	LoadGasPerUnit           = 18  // at least 1: see gas_per_unit.go
)

// LoadUnits is what loading one unpinned code of the given conservative
// compiled size costs, in budget units.
func LoadUnits(compiled uint64) uint64 {
	return LoadUnitsBase + compiled/LoadCompiledBytesPerUnit
}

// SetupCharger is a Budget that also charges loading a code before a call:
// ChargeSetup takes units from the block's budget and gas (SDK gas at the
// base price, scaled by the price in force) from ctx's meter; it may refuse
// (an error) or abort the transaction (BudgetAbort).
//
// The charger may charge a code's load once per block: checksum names the
// code and compiled is its conservative compiled size,
// for bounding what a block may load before repeats are charged again.
type SetupCharger interface {
	ChargeSetup(ctx sdk.Context, checksum []byte, units, gas, compiled uint64) error
}

// WithLoadPricing prices loading unpinned code (wasm-code-load-priced) while
// on(ctx) is true; pinned reports whether a checksum is pinned in ctx's
// state (pinned codes stay in memory and are exempt).
func (f *Filter) WithLoadPricing(on func(sdk.Context) bool, pinned func(ctx sdk.Context, checksum []byte) bool) *Filter {
	f.loadPriced, f.pinned = on, pinned
	return f
}

// LoadCost is what loading an unpinned code costs (budget units and gas at
// the base price), from the filter's verdict for its checksum; code reads
// the code's bytes when the verdict is not cached. False when the code
// cannot be read.
func (f *Filter) LoadCost(checksum []byte, code func() ([]byte, error)) (units, gas uint64, ok bool) {
	known, v := f.verdictFor(checksum, code)
	if !known {
		return 0, 0, false
	}
	units = LoadUnits(v.Compiled)
	return units, units * LoadGasPerUnit, true
}

// chargeLoad charges loading the code of one call, before it runs.
func (e *Engine) chargeLoad(checksum wasmvm.Checksum, querier wasmvm.Querier) error {
	if e.f.loadPriced == nil {
		return nil
	}
	sc, ok := e.f.budget.(SetupCharger)
	if !ok {
		return nil
	}
	ctx, ok := contextOf(querier)
	if !ok || !e.f.loadPriced(ctx) {
		return nil
	}
	if e.f.pinned != nil && e.f.pinned(ctx, checksum) {
		return nil
	}
	known, v := e.verdict(checksum)
	if !known {
		return nil
	}
	units := LoadUnits(v.Compiled)
	return sc.ChargeSetup(ctx, checksum, units, units*LoadGasPerUnit, v.Compiled)
}

// WithUploadBounds turns on the upload ordering and bounds of
// msgServer.checkUpload while on(ctx) is true; permitted reports whether
// wasmd's upload access allows a sender (nil allows everyone).
func (f *Filter) WithUploadBounds(on func(sdk.Context) bool, permitted func(ctx sdk.Context, sender string) bool) *Filter {
	f.uploadBounds, f.uploadPermitted = on, permitted
	return f
}

// Budget limits the VM gas a block's calls may use together (see
// app/wasm_block_budget.go). Limit returns the gas limit to give one VM call
// in ctx (at most gasLimit), whether that limit is below gasLimit (the call
// was cut by the budget), or an error when nothing is left; Used records what
// the call used inside the VM and outside it (gasExternal: storage, queries,
// nested calls, read off the meter around the call), with what Limit
// returned. Both are called only with the
// context wasmd hands the VM, and a Budget decides itself which contexts it
// counts. Either may panic with BudgetAbort to abort the whole transaction.
type Budget interface {
	Limit(ctx sdk.Context, gasLimit uint64) (limit uint64, clamped bool, err error)
	Used(ctx sdk.Context, gasUsed, gasExternal, limit uint64, clamped bool)
}

// consumed reads the gas meter wasmd hands the VM (in VM gas), 0 without
// one. What it grows by during a call is the call's external gas: storage
// reads and writes, query results and nested calls, which the VM does not
// count in the gas it reports.
func consumed(m wasmvm.GasMeter) uint64 {
	if m == nil {
		return 0
	}
	return m.GasConsumed()
}

// UploadCharger is a Budget that also charges storing (compiling) code:
// ChargeUpload is called with the upload's units (its uncompressed size, or
// from wasm-upload-compiled-size the larger of that and its conservative
// compiled size over CompiledBytesPerUploadUnit) before the code is stored,
// and may refuse it (an error) or abort the transaction (BudgetAbort).
type UploadCharger interface {
	ChargeUpload(ctx sdk.Context, units int) error
}

// UploadFitter is an UploadCharger that can also tell, without taking
// anything, whether an upload of the given units would be charged now: an
// error (or a BudgetAbort panic) exactly when ChargeUpload would refuse it.
// The message server asks before charging an upload's storage gas, so an
// upload the block cannot compile is refused cheaply.
type UploadFitter interface {
	UploadFits(ctx sdk.Context, units int) error
}

// BudgetAbort is the panic a Budget raises to abort a whole transaction
// whose contract work the block's budget refused or cut. It is not
// storetypes.ErrorOutOfGas on purpose: wasmd recovers that one in a
// submessage with its own gas limit, and an IBC packet receive turns an
// ordinary error into a permanent error acknowledgement, while an aborted
// transaction is simply retried later. The application turns it into an
// ordinary transaction failure (a run-tx recovery handler).
type BudgetAbort struct {
	Reason string
}

func (a BudgetAbort) String() string { return "block wasm gas budget: " + a.Reason }

// IsBudgetAbort reports whether a recovered panic value is a BudgetAbort.
func IsBudgetAbort(r any) bool {
	_, ok := r.(BudgetAbort)
	return ok
}

// WithBudget makes every VM call go through b.
func (f *Filter) WithBudget(b Budget) *Filter {
	f.budget = b
	return f
}

// budgetLimit is the gas limit for one VM call under the budget.
func (e *Engine) budgetLimit(querier wasmvm.Querier, gasLimit uint64) (uint64, bool, error) {
	if e.f.budget == nil {
		return gasLimit, false, nil
	}
	ctx, ok := contextOf(querier)
	if !ok {
		return gasLimit, false, nil
	}
	return e.f.budget.Limit(ctx, gasLimit)
}

// budgetUsed records one VM call's gas under the budget.
func (e *Engine) budgetUsed(querier wasmvm.Querier, gasUsed, gasExternal, limit uint64, clamped bool) {
	if e.f.budget == nil {
		return
	}
	if ctx, ok := contextOf(querier); ok {
		e.f.budget.Used(ctx, gasUsed, gasExternal, limit, clamped)
	}
}

// WithDeserCostScale makes every VM call price the JSON deserialisation of
// its result at deserCost * num() / den(). wasmd sets that price in VM gas
// per byte at its default multiplier; an application that converts SDK gas
// to VM gas at another multiplier scales it so that a result byte costs the
// same SDK gas as before.
func (f *Filter) WithDeserCostScale(scale func() (num, den uint64)) *Filter {
	f.deserScale = scale
	return f
}

// deser is deserCost under the filter's scale.
func (e *Engine) deser(c wasmvmtypes.UFraction) wasmvmtypes.UFraction {
	if e.f.deserScale == nil {
		return c
	}
	num, den := e.f.deserScale()
	if num == den || den == 0 {
		return c
	}
	return wasmvmtypes.UFraction{Numerator: c.Numerator * num, Denominator: c.Denominator * den}
}

// WithFloatRefusal makes the filter also refuse, while both on(ctx) and
// noFloat(ctx) are true, code that contains float operators (CheckFloats):
// storing it, and every call that would run it. noFloat must read consensus
// state only and must not charge gas.
func (f *Filter) WithFloatRefusal(noFloat func(sdk.Context) bool) *Filter {
	f.noFloat = noFloat
	return f
}

// refusesFloats reports whether float operators are refused in ctx.
func (f *Filter) refusesFloats(ctx sdk.Context) bool {
	return f.noFloat != nil && f.on(ctx) && f.noFloat(ctx)
}

// WithShapeBounds makes the filter also refuse, while both on(ctx) and
// shape(ctx) are true, code over the module shape bounds (CheckShape):
// storing it, and every call that would run it. shape must read consensus
// state only and must not charge gas.
func (f *Filter) WithShapeBounds(shape func(sdk.Context) bool) *Filter {
	f.shape = shape
	return f
}

// refusesShape reports whether the module shape bounds apply in ctx.
func (f *Filter) refusesShape(ctx sdk.Context) bool {
	return f.shape != nil && f.on(ctx) && f.shape(ctx)
}

// New returns a filter that refuses only while on(ctx) is true. on must read
// consensus state only and must not charge gas.
func New(on func(sdk.Context) bool) *Filter {
	return &Filter{on: on, verdicts: NewVerdicts()}
}

// DecorateEngine wraps wasmd's VM (wasmkeeper.WithWasmEngineDecorator).
func (f *Filter) DecorateEngine(inner wasmtypes.WasmEngine) wasmtypes.WasmEngine {
	return &Engine{WasmEngine: inner, f: f}
}

// Engine refuses every entry point that runs contract code (instantiate,
// execute, migrate, sudo, reply, smart query and the IBC entry points) when the
// code fails Check and the switch is on in the calling context. Storing,
// unchecked storing (genesis import and state-sync restore), pinning, analysis
// and GetCode are passed through untouched: a node must always be able to hold
// and restore every code the chain has, whether or not it may run.
//
// For code that passes Check the engine adds nothing to the call itself: no
// state read, no gas, no change in results, whether the switch is on or off.
// When the application gives the filter a Budget, every call's VM gas limit
// is capped by it and its use recorded (the per-block wasm budget).
type Engine struct {
	wasmtypes.WasmEngine
	f *Filter
}

var _ wasmtypes.WasmEngine = (*Engine)(nil)

// verdict returns the filter's verdict on a stored code, from the cache or by
// reading the code's bytes from the VM.
func (e *Engine) verdict(checksum wasmvm.Checksum) (known bool, v Verdict) {
	return e.f.verdictFor(checksum, func() ([]byte, error) { return e.WasmEngine.GetCode(checksum) })
}

// verdictFor is the cached verdict for a checksum, computed from code() on a
// miss. A code that cannot be read is not cached (the bytes may arrive, as
// in a state-sync restore).
func (f *Filter) verdictFor(checksum []byte, code func() ([]byte, error)) (bool, Verdict) {
	if ok, v := f.verdicts.Get(checksum); ok {
		return true, v
	}
	bz, err := code()
	if err != nil {
		return false, Verdict{}
	}
	st, cerr := Inspect(bz)
	v := Verdict{Err: cerr, Size: len(bz), Functions: st.Functions, Compiled: ConservativeCompiledSize(len(bz), st), SetupUnits: CallSetupUnits(st), MemoryPages: uint64(st.MemoryPages)}
	if v.Err == nil && st.FloatOps > 0 {
		v.Floats = fmt.Errorf("%w: %d float operators", ErrFloatOpcode, st.FloatOps)
	}
	if v.Err == nil {
		v.Shape = CheckShape(st)
	}
	f.verdicts.Put(checksum, v)
	return true, v
}

// refuse decides one call. The switch is read only for a failing code, from the
// sdk.Context wasmd hands the VM inside its querier. A failing code reached
// without that context is refused: it cannot be shown that the switch is off.
// Code with float operators fails only while the float switch is on too, and
// code over the module shape bounds only while their switch is on too.
func (e *Engine) refuse(checksum wasmvm.Checksum, querier wasmvm.Querier) error {
	known, v := e.verdict(checksum)
	if !known {
		return nil
	}
	reason := v.Err
	if reason == nil {
		for _, c := range []struct {
			err error
			set bool // the switch is wired at all
			on  func(sdk.Context) bool
		}{
			{v.Floats, e.f.noFloat != nil, e.f.refusesFloats},
			{v.Shape, e.f.shape != nil, e.f.refusesShape},
		} {
			if c.err == nil || !c.set {
				continue
			}
			ctx, ok := contextOf(querier)
			if ok && !c.on(ctx) {
				continue
			}
			return errorsmod.Wrapf(ErrCodeRefused, "code %X: %v", []byte(checksum), c.err)
		}
		return nil
	}
	ctx, ok := contextOf(querier)
	if ok && !e.f.on(ctx) {
		return nil
	}
	return errorsmod.Wrapf(ErrCodeRefused, "code %X: %v", []byte(checksum), reason)
}

func contextOf(q wasmvm.Querier) (sdk.Context, bool) {
	switch h := q.(type) {
	case wasmkeeper.QueryHandler:
		return h.Ctx, h.Ctx.MultiStore() != nil
	case *wasmkeeper.QueryHandler:
		if h != nil {
			return h.Ctx, h.Ctx.MultiStore() != nil
		}
	}
	return sdk.Context{}, false
}

func (e *Engine) Instantiate(checksum wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Instantiate(checksum, env, info, initMsg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) Execute(checksum wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Execute(checksum, env, info, executeMsg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) Query(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Query(checksum, env, queryMsg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) Migrate(checksum wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Migrate(checksum, env, migrateMsg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) MigrateWithInfo(checksum wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, migrateInfo wasmvmtypes.MigrateInfo, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.MigrateWithInfo(checksum, env, migrateMsg, migrateInfo, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) Sudo(checksum wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Sudo(checksum, env, sudoMsg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) Reply(checksum wasmvm.Checksum, env wasmvmtypes.Env, reply wasmvmtypes.Reply, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Reply(checksum, env, reply, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCChannelOpen(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCChannelOpenResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCChannelOpen(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCChannelConnect(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelConnectMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCChannelConnect(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCChannelClose(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelCloseMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCChannelClose(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCPacketReceive(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketReceiveMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCPacketReceive(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCPacketAck(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketAckMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCPacketAck(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCPacketTimeout(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketTimeoutMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCPacketTimeout(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCSourceCallback(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCSourceCallbackMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCSourceCallback(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

func (e *Engine) IBCDestinationCallback(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCDestinationCallbackMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	if err := e.chargeLoad(checksum, querier); err != nil {
		return nil, 0, err
	}
	q, cs, err := e.chargeCallSetup(checksum, querier)
	if err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCDestinationCallback(checksum, env, msg, store, goapi, q, gasMeter, limit, e.deser(deserCost))
	if err = e.afterCall(querier, cs, used, consumed(gasMeter)-ext0, limit, clamped, err); err != nil {
		return nil, used, err
	}
	return res, used, nil
}

// StoreCode prices compiling at wasmd's own rate whatever multiplier the
// application converts SDK gas at (the calibrated price, a congestion
// factor): the VM charges compiling in VM gas at wasmd's default
// multiplier, and both the VM's limit and the gas it reports are rescaled
// so that a byte costs the same SDK gas as at that default (3). Compiling
// does not get slower with congestion, and the block wasm budget already
// bounds its CPU (by size); pricing it at the calibrated multiplier would
// only make ordinary contracts too large to store. Without a scale the call
// is passed through.
func (e *Engine) StoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvm.Checksum, uint64, error) {
	num, den, ok := e.uploadScale()
	if !ok {
		return e.WasmEngine.StoreCode(code, gasLimit)
	}
	cs, used, err := e.WasmEngine.StoreCode(code, rescale(gasLimit, den, num))
	return cs, rescaleUp(used, num, den), err
}

// SimulateStoreCode is StoreCode for simulation.
func (e *Engine) SimulateStoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvm.Checksum, uint64, error) {
	num, den, ok := e.uploadScale()
	if !ok {
		return e.WasmEngine.SimulateStoreCode(code, gasLimit)
	}
	cs, used, err := e.WasmEngine.SimulateStoreCode(code, rescale(gasLimit, den, num))
	return cs, rescaleUp(used, num, den), err
}

// uploadScale is the multiplier in force over wasmd's default (num/den), or
// false when there is none or it is the default.
func (e *Engine) uploadScale() (num, den uint64, ok bool) {
	if e.f.deserScale == nil {
		return 0, 0, false
	}
	num, den = e.f.deserScale()
	if num == 0 || den == 0 || num == den {
		return 0, 0, false
	}
	return num, den, true
}

// rescale is v*mul/div, saturating at the maximum uint64.
func rescale(v, mul, div uint64) uint64 {
	r := new(big.Int).Mul(new(big.Int).SetUint64(v), new(big.Int).SetUint64(mul))
	r.Quo(r, new(big.Int).SetUint64(div))
	if !r.IsUint64() {
		return math.MaxUint64
	}
	return r.Uint64()
}

// rescaleUp is v*mul/div rounded up, saturating.
func rescaleUp(v, mul, div uint64) uint64 {
	r := new(big.Int).Mul(new(big.Int).SetUint64(v), new(big.Int).SetUint64(mul))
	r.Add(r, new(big.Int).SetUint64(div-1))
	r.Quo(r, new(big.Int).SetUint64(div))
	if !r.IsUint64() {
		return math.MaxUint64
	}
	return r.Uint64()
}
