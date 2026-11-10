package wasmfilter

import (
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
	on       func(sdk.Context) bool
	verdicts *Verdicts
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
// For code that passes Check the engine adds nothing: no state read, no gas,
// no change in results, whether the switch is on or off.
type Engine struct {
	wasmtypes.WasmEngine
	f *Filter
}

var _ wasmtypes.WasmEngine = (*Engine)(nil)

// verdict returns Check's result for a stored code, from the cache or by
// reading the code's bytes from the VM.
func (e *Engine) verdict(checksum wasmvm.Checksum) (known bool, err error) {
	if ok, v := e.f.verdicts.Get(checksum); ok {
		return true, v
	}
	code, gerr := e.WasmEngine.GetCode(checksum)
	if gerr != nil {
		// The call below fails on its own for a code the node does not hold;
		// do not cache, the bytes may arrive (state-sync restore).
		return false, nil
	}
	v := Check(code)
	e.f.verdicts.Put(checksum, v)
	return true, v
}

// refuse decides one call. The switch is read only for a failing code, from the
// sdk.Context wasmd hands the VM inside its querier. A failing code reached
// without that context is refused: it cannot be shown that the switch is off.
func (e *Engine) refuse(checksum wasmvm.Checksum, querier wasmvm.Querier) error {
	known, v := e.verdict(checksum)
	if !known || v == nil {
		return nil
	}
	ctx, ok := contextOf(querier)
	if ok && !e.f.on(ctx) {
		return nil
	}
	return errorsmod.Wrapf(ErrCodeRefused, "code %X: %v", []byte(checksum), v)
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
	return e.WasmEngine.Instantiate(checksum, env, info, initMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) Execute(checksum wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.Execute(checksum, env, info, executeMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) Query(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.Query(checksum, env, queryMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) Migrate(checksum wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.Migrate(checksum, env, migrateMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) MigrateWithInfo(checksum wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, migrateInfo wasmvmtypes.MigrateInfo, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.MigrateWithInfo(checksum, env, migrateMsg, migrateInfo, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) Sudo(checksum wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.Sudo(checksum, env, sudoMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) Reply(checksum wasmvm.Checksum, env wasmvmtypes.Env, reply wasmvmtypes.Reply, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.Reply(checksum, env, reply, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCChannelOpen(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCChannelOpenResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCChannelOpen(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCChannelConnect(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelConnectMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCChannelConnect(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCChannelClose(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCChannelCloseMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCChannelClose(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCPacketReceive(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketReceiveMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCPacketReceive(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCPacketAck(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketAckMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCPacketAck(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCPacketTimeout(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCPacketTimeoutMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCPacketTimeout(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCSourceCallback(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCSourceCallbackMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCSourceCallback(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (e *Engine) IBCDestinationCallback(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg wasmvmtypes.IBCDestinationCallbackMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	if err := e.refuse(checksum, querier); err != nil {
		return nil, 0, err
	}
	return e.WasmEngine.IBCDestinationCallback(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}
