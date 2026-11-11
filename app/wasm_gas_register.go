package app

import (
	"sync/atomic"

	abci "github.com/cometbft/cometbft/abci/types"

	storetypes "cosmossdk.io/store/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// CalibratedWasmGasMultiplier is how many units of CosmWasm VM gas one unit of
// SDK gas buys from the wasm-gas-calibrated switch (wasmd's default is
// wasmtypes.DefaultGasMultiplier, 140,000). The VM meters every operator at a
// flat price. Measured (app TestWasmGasTiming, QORE_WASM_TIMING=1) against an
// EVM loop reached through x/crossvm (1.66 ns per gas on the measuring host):
// at the default price ordinary compiled contract code (wasmd's hackatom
// cpu_loop) cost 48 ns per gas, 29 times the EVM; at 5,000 it costs 1.9 ns
// per gas, about the EVM's. Real contract calls are dominated by storage and
// were already below the EVM per gas (cw20 transfer, cw4, cw721 mint, cw3
// propose: 0.8 to 1.4 ns per gas); at 5,000 they cost 11 to 16 percent more
// gas (cw20 transfer 76,135 -> 87,278). The slowest operator classes stay
// above the EVM per gas (memory.grow about 50x, integer rem/div 11 to 18x);
// float operators, the slowest of all, are refused by the code filter
// (wasm-float-ops-refused), and the per-transaction cap (cosmos-tx-gas-cap)
// and the block gas limit bound what any of them can do.
const CalibratedWasmGasMultiplier uint64 = 5_000

// calibratedGasRegister is wasmd's default gas register with the
// SDK-to-VM gas conversion read from a multiplier the application sets at the
// start of every block from consensus state (syncWasmGasMultiplier). wasmd
// asks the register without a context, so the switch cannot be read where the
// conversion happens; setting it in PreBlock, before any transaction of the
// block runs, gives every node the same value for the whole block, and a node
// replaying history gets the value of each block's own state.
type calibratedGasRegister struct {
	wasmtypes.WasmGasRegister
	mult atomic.Uint64
}

func newCalibratedGasRegister() *calibratedGasRegister {
	g := &calibratedGasRegister{WasmGasRegister: wasmtypes.NewDefaultWasmGasRegister()}
	g.mult.Store(wasmtypes.DefaultGasMultiplier)
	return g
}

// ToWasmVMGas converts SDK gas to VM gas, as wasmd's register does, at the
// multiplier in force.
func (g *calibratedGasRegister) ToWasmVMGas(source storetypes.Gas) uint64 {
	m := g.mult.Load()
	x := source * m
	if m != 0 && x/m != source {
		panic(storetypes.ErrorGasOverflow{Descriptor: "ToWasmVMGas"})
	}
	return x
}

// FromWasmVMGas converts VM gas to SDK gas at the multiplier in force.
func (g *calibratedGasRegister) FromWasmVMGas(source uint64) storetypes.Gas {
	return source / g.mult.Load()
}

// Multiplier is the conversion in force.
func (g *calibratedGasRegister) Multiplier() uint64 { return g.mult.Load() }

// wasmGasRegister returns the app's one gas register.
func (app *QoreChainApp) wasmGasRegister() *calibratedGasRegister {
	if app.gasRegister == nil {
		app.gasRegister = newCalibratedGasRegister()
	}
	return app.gasRegister
}

// syncWasmGasMultiplier sets the multiplier from the wasm-gas-calibrated
// switch in ctx's state. The feature read is free of gas.
func (app *QoreChainApp) syncWasmGasMultiplier(ctx sdk.Context) {
	m := wasmtypes.DefaultGasMultiplier
	if app.PQCKeeper != nil && app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureWasmGasCalibrated) {
		m = CalibratedWasmGasMultiplier
	}
	app.wasmGasRegister().mult.Store(m)
}

// preBlockerSyncingWasmGas is the module manager's PreBlock followed by
// setting the wasm gas multiplier for the block about to run. It runs after
// the module PreBlockers so that a release upgrade that turns the switch on
// prices the transactions of its own block the new way, like every other
// switch the upgrade enables.
func (app *QoreChainApp) preBlockerSyncingWasmGas(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	res, err := app.App.PreBlocker(ctx, req)
	app.syncWasmGasMultiplier(ctx)
	// The block's CosmWasm execution budget starts full (wasm_block_budget.go).
	app.resetWasmBlockBudget(ctx)
	return res, err
}

// deserCostScale is the factor wasmd's JSON deserialisation price (VM gas
// per result byte, set at its default multiplier) is scaled by, so that a
// result byte costs the same SDK gas at the multiplier in force as at
// wasmd's default (one SDK gas per byte). Identity at the default.
func (g *calibratedGasRegister) deserCostScale() (num, den uint64) {
	return g.mult.Load(), wasmtypes.DefaultGasMultiplier
}
