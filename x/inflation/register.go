//go:build proprietary

package inflation

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/inflation/keeper"
	"github.com/qorechain/qorechain-core/x/inflation/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the InflationKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                              { return a.k.Logger() }
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params          { return a.k.GetParams(ctx) }
func (a *keeperAdapter) SetParams(ctx sdk.Context, p types.Params) error { return a.k.SetParams(ctx, p) }
func (a *keeperAdapter) GetCurrentInflationRate(ctx sdk.Context) math.LegacyDec {
	return a.k.GetCurrentInflationRate(ctx)
}
func (a *keeperAdapter) GetEpochInfo(ctx sdk.Context) types.EpochInfo { return a.k.GetEpochInfo(ctx) }
func (a *keeperAdapter) MintEpochEmission(ctx sdk.Context) error      { return a.k.MintEpochEmission(ctx) }
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	a.k.InitGenesis(ctx, gs)
}
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return a.k.ExportGenesis(ctx)
}

// RealNewInflationKeeper creates the real inflation keeper wrapped in an adapter.
// The bankKeeper parameter must satisfy the keeper.BankKeeper interface.
func RealNewInflationKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper keeper.BankKeeper, logger log.Logger) InflationKeeper {
	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real inflation AppModule.
func RealNewAppModule(k InflationKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("InflationKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
