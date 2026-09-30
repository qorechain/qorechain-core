package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// upgradeV3_2_2 is the v3.2.2 release handler. It runs once, at the plan height,
// on every node. Each fix's one-time state work lives in its own file
// (upgrade_v3_2_2_<area>.go) and runs here before the switches turn on, so a
// fix never sees its switch on and its state unprepared.
func (app *QoreChainApp) upgradeV3_2_2(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, step := range []func(sdk.Context) error{
		app.upgradeV3_2_2EVM7702,
		app.upgradeV3_2_2StakingAdmission,
		app.upgradeV3_2_2AMM,
		app.upgradeV3_2_2AbstractAccount,
		app.upgradeV3_2_2EVMMisc,
		app.upgradeV3_2_2Modules,
		app.upgradeV3_2_2RDKSVM,
		app.upgradeV3_2_2AnteParity,
	} {
		if err := step(sdkCtx); err != nil {
			return fromVM, err
		}
	}
	for _, f := range pqctypes.ReleaseV3_2_2ProtocolFeatures {
		if err := app.PQCKeeper.EnableProtocolFeature(sdkCtx, f); err != nil {
			return fromVM, err
		}
	}
	return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
}
