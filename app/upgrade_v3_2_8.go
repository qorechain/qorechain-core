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
//
// Step order: the release features are enabled first, because later steps
// check them (the SVM zero-lamport report and the custody liability start).
// Then the params writes: the SVM lane params, the crossvm bridge target pins,
// and the removal of consent grants inside authz. The SVM store steps follow:
// bytecode accounts move to the loader, zero-lamport accounts are reported,
// and the custody liability is started last, from the store as the earlier
// SVM steps leave it.
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
	if err := app.upgradeV3_2_8RemoveConsentGrants(sdkCtx); err != nil {
		return fromVM, err
	}
	if err := app.upgradeV3_2_8SVMBytecodeOwners(sdkCtx); err != nil {
		return fromVM, err
	}
	app.upgradeV3_2_8SVMZeroLamportReport(sdkCtx)
	app.upgradeV3_2_8SVMCustodyLiability(sdkCtx)
	return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
}

// svmBytecodeMigrator is implemented by the full SVM keeper.
type svmBytecodeMigrator interface {
	MigrateBytecodeOwnersV328(ctx sdk.Context) (int, error)
}

// upgradeV3_2_8SVMBytecodeOwners hands the bytecode account of every program
// deployed before svm-bytecode-loader-owned to the loader, so from the plan
// height no program owns its own code (FeatureSVMBytecodeLoaderOwned). It runs
// on every chain; a chain with no deployed program writes nothing. The stub
// keeper of a build without SVM has no programs and no migration.
func (app *QoreChainApp) upgradeV3_2_8SVMBytecodeOwners(ctx sdk.Context) error {
	m, ok := app.SVMKeeper.(svmBytecodeMigrator)
	if !ok {
		return nil
	}
	moved, err := m.MigrateBytecodeOwnersV328(ctx)
	if err != nil {
		return err
	}
	if moved > 0 {
		app.Logger().Info("v3.2.8: SVM bytecode accounts moved to the loader", "count", moved)
	}
	return nil
}

// svmZeroLamportReporter is implemented by the full SVM keeper.
type svmZeroLamportReporter interface {
	ReportZeroLamportAccountsV328(ctx sdk.Context) int
}

// upgradeV3_2_8SVMZeroLamportReport lists, as svm_zero_lamport_account events in
// store order, the stored accounts other than wallets and programs that already
// hold zero lamports. They are left exactly as they are: no deletion and no
// funding (svm-zero-lamport-touched-only never deletes an account an
// instruction did not change).
func (app *QoreChainApp) upgradeV3_2_8SVMZeroLamportReport(ctx sdk.Context) {
	r, ok := app.SVMKeeper.(svmZeroLamportReporter)
	if !ok || !app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureSVMZeroLamportTouchedOnly) {
		return
	}
	if n := r.ReportZeroLamportAccountsV328(ctx); n > 0 {
		app.Logger().Info("v3.2.8: SVM accounts holding zero lamports left in place", "count", n)
	}
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
	// svm-custody-hard-ceiling: the absolute ceiling is written with the
	// threshold; it is inert until that feature is enabled with the release.
	params.MaxCustodyHardUqor = pqctypes.SVMLaneGovernedInitialCustodyHardCeilingUqor
	// svm-account-excess-cap: one account holds at most 10,000 QOR above rent,
	// so pinning the ceiling takes at least 100 funded accounts.
	params.MaxAccountExcessUqor = pqctypes.SVMLaneGovernedInitialAccountExcessUqor
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

// svmCustodyLiabilityInit is implemented by the full SVM keeper.
type svmCustodyLiabilityInit interface {
	InitCustodyLiabilityV328(ctx sdk.Context)
}

// upgradeV3_2_8SVMCustodyLiability starts the custody liability accounting of
// svm-custody-liability-window from the store as it stands after the other
// SVM steps of this handler, so the cap reads what the SVM owes from the first
// block. A module balance above the cap (faucet pre-funding, donations) is not
// a liability and blocks nothing.
func (app *QoreChainApp) upgradeV3_2_8SVMCustodyLiability(ctx sdk.Context) {
	l, ok := app.SVMKeeper.(svmCustodyLiabilityInit)
	if !ok || !app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureSVMCustodyLiabilityWindow) {
		return
	}
	l.InitCustodyLiabilityV328(ctx)
}
