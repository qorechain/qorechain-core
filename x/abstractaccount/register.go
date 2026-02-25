//go:build proprietary

package abstractaccount

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/abstractaccount/keeper"
	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the AbstractAccountKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                                                    { return a.k.Logger() }
func (a *keeperAdapter) GetConfig(ctx sdk.Context) types.AbstractAccountConfig                 { return a.k.GetConfig(ctx) }
func (a *keeperAdapter) SetConfig(ctx sdk.Context, c types.AbstractAccountConfig) error        { return a.k.SetConfig(ctx, c) }
func (a *keeperAdapter) IsEnabled(ctx sdk.Context) bool                                        { return a.k.IsEnabled(ctx) }
func (a *keeperAdapter) GetAccount(ctx sdk.Context, addr string) (types.AbstractAccount, bool) { return a.k.GetAccount(ctx, addr) }
func (a *keeperAdapter) SetAccount(ctx sdk.Context, acc types.AbstractAccount) error           { return a.k.SetAccount(ctx, acc) }
func (a *keeperAdapter) GetAllAccounts(ctx sdk.Context) []types.AbstractAccount                { return a.k.GetAllAccounts(ctx) }
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState)                    { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState                     { return a.k.ExportGenesis(ctx) }

// RealNewAbstractAccountKeeper creates the real abstract account keeper wrapped in an adapter.
func RealNewAbstractAccountKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) AbstractAccountKeeper {
	k := keeper.NewKeeper(cdc, storeKey, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real abstractaccount AppModule.
func RealNewAppModule(k AbstractAccountKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("AbstractAccountKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
