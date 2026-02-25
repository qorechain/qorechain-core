//go:build proprietary

package gasabstraction

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/gasabstraction/keeper"
	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the GasAbstractionKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                                                 { return a.k.Logger() }
func (a *keeperAdapter) GetConfig(ctx sdk.Context) types.GasAbstractionConfig               { return a.k.GetConfig(ctx) }
func (a *keeperAdapter) SetConfig(ctx sdk.Context, c types.GasAbstractionConfig) error      { return a.k.SetConfig(ctx, c) }
func (a *keeperAdapter) IsEnabled(ctx sdk.Context) bool                                     { return a.k.IsEnabled(ctx) }
func (a *keeperAdapter) GetAcceptedTokens(ctx sdk.Context) []types.AcceptedFeeToken         { return a.k.GetAcceptedTokens(ctx) }
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState)                 { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState                  { return a.k.ExportGenesis(ctx) }

// RealNewGasAbstractionKeeper creates the real gas abstraction keeper wrapped in an adapter.
func RealNewGasAbstractionKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) GasAbstractionKeeper {
	k := keeper.NewKeeper(cdc, storeKey, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real gasabstraction AppModule.
func RealNewAppModule(k GasAbstractionKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("GasAbstractionKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
