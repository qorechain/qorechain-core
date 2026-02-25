//go:build proprietary

package burn

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/burn/keeper"
	"github.com/qorechain/qorechain-core/x/burn/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the BurnKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                              { return a.k.Logger() }
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params          { return a.k.GetParams(ctx) }
func (a *keeperAdapter) SetParams(ctx sdk.Context, p types.Params) error { return a.k.SetParams(ctx, p) }
func (a *keeperAdapter) BurnFromSource(ctx sdk.Context, source types.BurnSource, amount math.Int, txHash string) error {
	return a.k.BurnFromSource(ctx, source, amount, txHash)
}
func (a *keeperAdapter) GetTotalBurned(ctx sdk.Context) math.Int          { return a.k.GetTotalBurned(ctx) }
func (a *keeperAdapter) GetBurnStats(ctx sdk.Context) types.BurnStats     { return a.k.GetBurnStats(ctx) }
func (a *keeperAdapter) GetBurnRecords(ctx sdk.Context, limit int) []types.BurnRecord {
	return a.k.GetBurnRecords(ctx, limit)
}
func (a *keeperAdapter) DistributeFees(ctx sdk.Context) error               { return a.k.DistributeFees(ctx) }
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState  { return a.k.ExportGenesis(ctx) }

// RealNewBurnKeeper creates the real burn keeper wrapped in an adapter.
// The bankKeeper parameter must satisfy the keeper.BankKeeper interface.
func RealNewBurnKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper keeper.BankKeeper, logger log.Logger) BurnKeeper {
	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real burn AppModule.
func RealNewAppModule(k BurnKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("BurnKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
