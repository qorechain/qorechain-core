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
	if err := app.upgradeV3_2_8RemoveConsentGrants(sdkCtx); err != nil {
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
