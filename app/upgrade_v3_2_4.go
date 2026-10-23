package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	stakingtiertypes "github.com/qorechain/qorechain-core/x/stakingtier/types"
)

// upgradeV3_2_4 is the v3.2.4 release handler. It runs once, at the plan height,
// on every node: the one-time state steps first, then the release's switches.
func (app *QoreChainApp) upgradeV3_2_4(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	app.upgradeV3_2_4StakingTiers(sdkCtx)
	for _, f := range pqctypes.ReleaseV3_2_4ProtocolFeatures {
		if err := app.PQCKeeper.EnableProtocolFeature(sdkCtx, f); err != nil {
			return fromVM, err
		}
	}
	return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
}

// upgradeV3_2_4StakingTiers initialises x/stakingtier at the plan height. The
// handler records the module in the version map without running migrations,
// so its InitGenesis never runs on a chain that upgrades into it; this writes
// the default state instead (parameters, and a pool whose settlement clock
// starts at this block), and materialises the module account so the first
// bonus mint finds a module account rather than a plain one. Idempotent: an
// existing record is left as it is.
func (app *QoreChainApp) upgradeV3_2_4StakingTiers(ctx sdk.Context) {
	if app.StakingTierKeeper == nil {
		return
	}
	if app.StakingTierKeeper.GetPool(ctx).LastSettleTime == 0 {
		app.StakingTierKeeper.InitGenesis(ctx, *stakingtiertypes.DefaultGenesisState())
	}
	app.AccountKeeper.GetModuleAccount(ctx, stakingtiertypes.ModuleName)
}
