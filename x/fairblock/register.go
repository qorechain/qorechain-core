//go:build proprietary

package fairblock

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/fairblock/keeper"
	"github.com/qorechain/qorechain-core/x/fairblock/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the FairBlockKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                                       { return a.k.Logger() }
func (a *keeperAdapter) GetConfig(ctx sdk.Context) types.FairBlockConfig          { return a.k.GetConfig(ctx) }
func (a *keeperAdapter) SetConfig(ctx sdk.Context, c types.FairBlockConfig) error { return a.k.SetConfig(ctx, c) }
func (a *keeperAdapter) IsEnabled(ctx sdk.Context) bool                           { return a.k.IsEnabled(ctx) }
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState)       { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState        { return a.k.ExportGenesis(ctx) }

// RealNewFairBlockKeeper creates the real fairblock keeper wrapped in an adapter.
func RealNewFairBlockKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) FairBlockKeeper {
	k := keeper.NewKeeper(cdc, storeKey, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real fairblock AppModule.
func RealNewAppModule(k FairBlockKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("FairBlockKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
