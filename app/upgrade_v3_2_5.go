package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// upgradeV3_2_5 is the v3.2.5 release handler. It runs once, at the plan height,
// on every node: each area's one-time state step first, then the release's
// switches.
func (app *QoreChainApp) upgradeV3_2_5(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	steps := []func(sdk.Context) error{
		app.upgradeV3_2_5GovBurn,
		app.upgradeV3_2_5AA,
		app.upgradeV3_2_5PQCKEM,
		app.upgradeV3_2_5EVM,
		app.upgradeV3_2_5CrossVM,
		app.upgradeV3_2_5RDK,
		app.upgradeV3_2_5Wasm,
	}
	for _, step := range steps {
		if err := step(sdkCtx); err != nil {
			return fromVM, err
		}
	}
	for _, f := range pqctypes.ReleaseV3_2_5ProtocolFeatures {
		if err := app.PQCKeeper.EnableProtocolFeature(sdkCtx, f); err != nil {
			return fromVM, err
		}
	}
	return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
}
