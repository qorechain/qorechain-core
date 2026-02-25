//go:build proprietary

package xqore

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	"github.com/qorechain/qorechain-core/x/xqore/keeper"
	"github.com/qorechain/qorechain-core/x/xqore/types"
)

// Compile-time assertion: keeperAdapter satisfies rlconsensus.TokenomicsKeeper,
// replacing NilTokenomicsKeeper with real xQORE balance lookups.
var _ rlconsensusmod.TokenomicsKeeper = (*keeperAdapter)(nil)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the XQOREKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                                     { return a.k.Logger() }
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params                 { return a.k.GetParams(ctx) }
func (a *keeperAdapter) SetParams(ctx sdk.Context, p types.Params) error        { return a.k.SetParams(ctx, p) }
func (a *keeperAdapter) GetXQOREBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int {
	return a.k.GetXQOREBalance(ctx, addr)
}
func (a *keeperAdapter) Lock(ctx sdk.Context, owner sdk.AccAddress, amount math.Int) error {
	return a.k.Lock(ctx, owner, amount)
}
func (a *keeperAdapter) Unlock(ctx sdk.Context, owner sdk.AccAddress, amount math.Int) error {
	return a.k.Unlock(ctx, owner, amount)
}
func (a *keeperAdapter) GetPosition(ctx sdk.Context, owner sdk.AccAddress) (types.XQOREPosition, bool) {
	return a.k.GetPosition(ctx, owner)
}
func (a *keeperAdapter) GetAllPositions(ctx sdk.Context) []types.XQOREPosition {
	return a.k.GetAllPositions(ctx)
}
func (a *keeperAdapter) GetTotalLocked(ctx sdk.Context) math.Int { return a.k.GetTotalLocked(ctx) }
func (a *keeperAdapter) GetTotalXQORESupply(ctx sdk.Context) math.Int {
	return a.k.GetTotalXQORESupply(ctx)
}
func (a *keeperAdapter) GetGovernanceMultiplier(ctx sdk.Context, addr sdk.AccAddress) math.LegacyDec {
	return a.k.GetGovernanceMultiplier(ctx, addr)
}
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	a.k.InitGenesis(ctx, gs)
}
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return a.k.ExportGenesis(ctx)
}

// RealNewXQOREKeeper creates the real xQORE keeper wrapped in an adapter.
func RealNewXQOREKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper keeper.BankKeeper, logger log.Logger) XQOREKeeper {
	k := keeper.NewKeeper(cdc, storeKey, bankKeeper, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real xQORE AppModule.
func RealNewAppModule(k XQOREKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("XQOREKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
