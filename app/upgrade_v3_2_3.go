package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// upgradeV3_2_3 is the v3.2.3 release handler. It runs once, at the plan height,
// on every node: the one-time state steps first, then the release's switches.
func (app *QoreChainApp) upgradeV3_2_3(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, step := range []func(sdk.Context) error{
		app.upgradeV3_2_3Rdk,
		app.upgradeV3_2_3StakingVesting,
		app.upgradeV3_2_3Econ,
	} {
		if err := step(sdkCtx); err != nil {
			return fromVM, err
		}
	}
	for _, f := range pqctypes.ReleaseV3_2_3ProtocolFeatures {
		if err := app.PQCKeeper.EnableProtocolFeature(sdkCtx, f); err != nil {
			return fromVM, err
		}
	}
	return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
}
