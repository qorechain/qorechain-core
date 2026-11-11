package app

import (
	"bytes"
	"encoding/binary"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	"github.com/qorechain/qorechain-core/x/wasmfilter"
)

// wasmCodeFilter returns the app's one code filter, behind the
// wasm-code-filter protocol feature. The same filter (and its per-checksum
// verdicts) serves the VM decorator on the keeper and the message server
// decorator on the module:
//
//   - storing: MsgStoreCode, MsgStoreAndInstantiateContract and
//     MsgStoreAndMigrateContract are checked in the message server, which is
//     also what a governance proposal's messages go through;
//   - running: every VM entry point that executes a stored code (instantiate,
//     execute, migrate, sudo, reply, smart query, IBC entry points, also when
//     reached from another contract, a submessage or the cross-VM bridge) is
//     checked in the VM decorator;
//   - never refused: StoreCodeUnchecked (genesis import and state-sync
//     restore), pinning, AnalyzeCode and GetCode.
func (app *QoreChainApp) wasmCodeFilter() *wasmfilter.Filter {
	if app.codeFilter == nil {
		app.codeFilter = wasmfilter.New(app.featureSwitch(pqctypes.FeatureWasmCodeFilter)).
			WithFloatRefusal(app.featureSwitch(pqctypes.FeatureWasmFloatOpsRefused)).
			WithBudget(app.wasmBlockBudget()).
			WithDeserCostScale(app.wasmGasRegister().deserCostScale).
			WithUploadBounds(app.featureSwitch(pqctypes.FeatureWasmUploadBounds), app.wasmUploadPermitted).
			WithLoadPricing(app.featureSwitch(pqctypes.FeatureWasmCodeLoadPriced), app.wasmCodePinned).
			WithCompiledSizeCharge(app.featureSwitch(pqctypes.FeatureWasmUploadCompiledSize)).
			WithShapeBounds(app.featureSwitch(pqctypes.FeatureWasmModuleShapeBounds))
	}
	return app.codeFilter
}

// wasmCodeFilterAppModule is the wasm module with its message server wrapped
// by the code filter.
type wasmCodeFilterAppModule struct {
	wasm.AppModule
	filter *wasmfilter.Filter
}

func (m wasmCodeFilterAppModule) RegisterServices(cfg module.Configurator) {
	m.AppModule.RegisterServices(wasmfilter.Configurator{Configurator: cfg, F: m.filter})
}

// wasmCodePinned reports whether a code with this checksum is pinned in
// ctx's state (x/wasm's pinned-code index, a short list kept by governance).
// Read off the gas meter: the answer decides a price, it is not itself
// priced.
func (app *QoreChainApp) wasmCodePinned(ctx sdk.Context, checksum []byte) bool {
	if app.wasmStoreKey == nil {
		return false
	}
	free := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	store := free.KVStore(app.wasmStoreKey)
	iter := storetypes.KVStorePrefixIterator(store, wasmtypes.PinnedCodeIndexPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()[len(wasmtypes.PinnedCodeIndexPrefix):]
		if len(key) != 8 {
			continue
		}
		if info := app.WasmKeeper.GetCodeInfo(free, binary.BigEndian.Uint64(key)); info != nil && bytes.Equal(info.CodeHash, checksum) {
			return true
		}
	}
	return false
}
