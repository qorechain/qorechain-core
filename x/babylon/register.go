//go:build proprietary

package babylon

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/babylon/keeper"
	"github.com/qorechain/qorechain-core/x/babylon/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the BabylonKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                                            { return a.k.Logger() }
func (a *keeperAdapter) GetConfig(ctx sdk.Context) types.BTCRestakingConfig            { return a.k.GetConfig(ctx) }
func (a *keeperAdapter) SetConfig(ctx sdk.Context, c types.BTCRestakingConfig) error   { return a.k.SetConfig(ctx, c) }
func (a *keeperAdapter) IsEnabled(ctx sdk.Context) bool                                { return a.k.IsEnabled(ctx) }
func (a *keeperAdapter) GetStakingPosition(ctx sdk.Context, id string) (types.BTCStakingPosition, bool) {
	return a.k.GetStakingPosition(ctx, id)
}
func (a *keeperAdapter) SetStakingPosition(ctx sdk.Context, pos types.BTCStakingPosition) error {
	return a.k.SetStakingPosition(ctx, pos)
}
func (a *keeperAdapter) GetAllPositions(ctx sdk.Context) []types.BTCStakingPosition { return a.k.GetAllPositions(ctx) }
func (a *keeperAdapter) GetCheckpoint(ctx sdk.Context, epoch uint64) (types.BTCCheckpoint, bool) {
	return a.k.GetCheckpoint(ctx, epoch)
}
func (a *keeperAdapter) SetCheckpoint(ctx sdk.Context, cp types.BTCCheckpoint) error { return a.k.SetCheckpoint(ctx, cp) }
func (a *keeperAdapter) GetCurrentEpoch(ctx sdk.Context) uint64                      { return a.k.GetCurrentEpoch(ctx) }
func (a *keeperAdapter) GetEpochSnapshot(ctx sdk.Context, epoch uint64) (types.BabylonEpochSnapshot, bool) {
	return a.k.GetEpochSnapshot(ctx, epoch)
}
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState  { return a.k.ExportGenesis(ctx) }

// RealNewBabylonKeeper creates the real babylon keeper wrapped in an adapter.
func RealNewBabylonKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) BabylonKeeper {
	k := keeper.NewKeeper(cdc, storeKey, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real babylon AppModule.
func RealNewAppModule(k BabylonKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("BabylonKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
