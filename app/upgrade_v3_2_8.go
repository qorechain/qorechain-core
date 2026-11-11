package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

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
	if err := app.SVMKeeper.SetParams(ctx, params); err != nil {
		return err
	}
	return app.PQCKeeper.EnableProtocolFeature(ctx, pqctypes.FeatureSVMLaneGoverned)
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
