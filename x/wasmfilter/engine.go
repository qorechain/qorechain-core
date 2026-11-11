package wasmfilter

import (
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
// ChargeUpload is called with the uncompressed module size before the code
// is stored, and may refuse it (an error) or abort the transaction
// (BudgetAbort).
type UploadCharger interface {
	ChargeUpload(ctx sdk.Context, wasmBytes int) error
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
	if ok, v := e.f.verdicts.Get(checksum); ok {
		return true, v
	}
	code, gerr := e.WasmEngine.GetCode(checksum)
	if gerr != nil {
		// The call below fails on its own for a code the node does not hold;
		// do not cache, the bytes may arrive (state-sync restore).
		return false, Verdict{}
	}
	v = Verdict{Err: Check(code)}
	if v.Err == nil {
		v.Floats = CheckFloats(code)
	}
	e.f.verdicts.Put(checksum, v)
	return true, v
}

// refuse decides one call. The switch is read only for a failing code, from the
// sdk.Context wasmd hands the VM inside its querier. A failing code reached
// without that context is refused: it cannot be shown that the switch is off.
// Code with float operators fails only while the float switch is on too.
func (e *Engine) refuse(checksum wasmvm.Checksum, querier wasmvm.Querier) error {
	known, v := e.verdict(checksum)
	if !known {
		return nil
	}
	reason := v.Err
	if reason == nil {
		if v.Floats == nil || e.f.noFloat == nil {
			return nil
		}
		ctx, ok := contextOf(querier)
		if ok && !e.f.refusesFloats(ctx) {
			return nil
		}
		return errorsmod.Wrapf(ErrCodeRefused, "code %X: %v", []byte(checksum), v.Floats)
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
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Instantiate(checksum, env, info, initMsg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) Execute(checksum wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Execute(checksum, env, info, executeMsg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) Query(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Query(checksum, env, queryMsg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) Migrate(checksum wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Migrate(checksum, env, migrateMsg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) MigrateWithInfo(checksum wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, migrateInfo wasmvmtypes.MigrateInfo, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.MigrateWithInfo(checksum, env, migrateMsg, migrateInfo, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) Sudo(checksum wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Sudo(checksum, env, sudoMsg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) Reply(checksum wasmvm.Checksum, env wasmvmtypes.Env, reply wasmvmtypes.Reply, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.Reply(checksum, env, reply, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCChannelOpen(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCChannelOpenResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCChannelOpen(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCChannelConnect(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelConnectMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCChannelConnect(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCChannelClose(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelCloseMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCChannelClose(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCPacketReceive(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketReceiveMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCPacketReceive(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCPacketAck(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketAckMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCPacketAck(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCPacketTimeout(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketTimeoutMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCPacketTimeout(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCSourceCallback(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCSourceCallbackMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCSourceCallback(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
}

func (e *Engine) IBCDestinationCallback(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCDestinationCallbackMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	limit, clamped, err := e.budgetLimit(querier, gasLimit)
	if err != nil {
		return nil, 0, err
	}
	ext0 := consumed(gasMeter)
	res, used, err := e.WasmEngine.IBCDestinationCallback(checksum, env, msg, store, goapi, querier, gasMeter, limit, e.deser(deserCost))
	e.budgetUsed(querier, used, consumed(gasMeter)-ext0, limit, clamped)
	return res, used, err
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
