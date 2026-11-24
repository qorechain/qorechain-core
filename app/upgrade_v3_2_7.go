package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// upgradeV3_2_7 is the v3.2.7 release handler. It runs once, at the plan height,
// on every node and enables the release's switches.
func (app *QoreChainApp) upgradeV3_2_7(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, f := range pqctypes.ReleaseV3_2_7ProtocolFeatures {
		if err := app.PQCKeeper.EnableProtocolFeature(sdkCtx, f); err != nil {
			return fromVM, err
		}
	}
	return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
}
