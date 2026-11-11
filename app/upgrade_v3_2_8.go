package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	crossvmtypes "github.com/qorechain/qorechain-core/x/crossvm/types"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// upgradeV3_2_8 is the v3.2.8 release handler. It runs once, at the plan height,
// on every node and enables the release's switches.
func (app *QoreChainApp) upgradeV3_2_8(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, f := range pqctypes.ReleaseV3_2_8ProtocolFeatures {
		if err := app.PQCKeeper.EnableProtocolFeature(sdkCtx, f); err != nil {
			return fromVM, err
		}
	}
	if err := app.upgradeV3_2_8SVMLaneGoverned(sdkCtx); err != nil {
		return fromVM, err
	}
	if err := app.upgradeV3_2_8PinBridgeTargets(sdkCtx); err != nil {
		return fromVM, err
	}
	return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
}

// upgradeV3_2_8SVMLaneGoverned hands the SVM lane switch to governance on the
// public testnet only (FeatureSVMLaneGoverned). On any other chain id it does
// nothing at all: no feature record, no params write, so mainnet keeps the lane
// closed by the compile-time switch exactly as before.
//
// On the testnet it first writes the stored SVM params with Enabled=false and
// the initial custody cap, then enables the feature. The live params store
// enabled=true, so enabling the feature without the write would open the lane
// in the very next block. GetParams is read while the feature is still off, so
// it returns the stored values with Enabled already forced false; only Enabled
// and the cap change.
func (app *QoreChainApp) upgradeV3_2_8SVMLaneGoverned(ctx sdk.Context) error {
	if ctx.ChainID() != pqctypes.SVMLaneGovernedChainID {
		return nil
	}
	params := app.SVMKeeper.GetParams(ctx)
	params.Enabled = false
	params.MaxCustodyUqor = pqctypes.SVMLaneGovernedInitialCustodyCapUqor
	if err := app.SVMKeeper.SetParams(ctx, params); err != nil {
		return err
	}
	return app.PQCKeeper.EnableProtocolFeature(ctx, pqctypes.FeatureSVMLaneGoverned)
}

// upgradeV3_2_8PinBridgeTargets carries the EVM bridge allowlist over to
// crossvm-bridge-code-pin: every entry whose contract exists is pinned to the
// code ID that contract runs at the upgrade height and to its code history
// length, so each target the bridge reached before still answers, and a later
// migration by a non-governance admin (even one that returns to the pinned
// code) no longer changes what it reaches. An entry naming no contract is
// removed, with a crossvm_bridge_target_removed event: left in place it would
// become callable as soon as someone instantiated a contract at that address.
// With an empty allowlist (mainnet) nothing is written.
func (app *QoreChainApp) upgradeV3_2_8PinBridgeTargets(ctx sdk.Context) error {
	params, changed, removed := app.CrossVMKeeper.GetParams(ctx).PinBridgeTargets(func(contract string) crossvmtypes.BridgeContractState {
		return app.bridgeContractState(ctx, contract)
	})
	if !changed {
		return nil
	}
	if err := app.CrossVMKeeper.SetParams(ctx, params); err != nil {
		return err
	}
	for _, entry := range removed {
		ctx.EventManager().EmitEvent(sdk.NewEvent(crossvmtypes.EventTypeBridgeTargetRemoved,
			sdk.NewAttribute(crossvmtypes.AttributeKeyTarget, entry),
			sdk.NewAttribute(crossvmtypes.AttributeKeyReason, "names no contract"),
		))
	}
	return nil
}

// bridgeContractState reads what the bridge code pin needs to know about a
// CosmWasm contract from the wasm keeper: its code, its admin and the length
// of its code history (one entry for instantiation, one per migration).
func (app *QoreChainApp) bridgeContractState(ctx sdk.Context, contract string) crossvmtypes.BridgeContractState {
	addr, err := sdk.AccAddressFromBech32(contract)
	if err != nil {
		return crossvmtypes.BridgeContractState{}
	}
	info := app.WasmKeeper.GetContractInfo(ctx, addr)
	if info == nil {
		return crossvmtypes.BridgeContractState{}
	}
	return crossvmtypes.BridgeContractState{
		Found: true, CodeID: info.CodeID, Admin: info.Admin,
		HistoryLen: uint64(len(app.WasmKeeper.GetContractHistory(ctx, addr))),
	}
}
