//go:build proprietary

package bridge

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	burnmod "github.com/qorechain/qorechain-core/x/burn"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	"github.com/qorechain/qorechain-core/x/bridge/keeper"
	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the BridgeKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger                                                    { return a.k.Logger() }
func (a *keeperAdapter) GetConfig(ctx sdk.Context) types.BridgeConfig                          { return a.k.GetConfig(ctx) }
func (a *keeperAdapter) SetConfig(ctx sdk.Context, c types.BridgeConfig) error                 { return a.k.SetConfig(ctx, c) }
func (a *keeperAdapter) GetChainConfig(ctx sdk.Context, id string) (types.ChainConfig, bool)   { return a.k.GetChainConfig(ctx, id) }
func (a *keeperAdapter) SetChainConfig(ctx sdk.Context, cc types.ChainConfig) error            { return a.k.SetChainConfig(ctx, cc) }
func (a *keeperAdapter) GetAllChainConfigs(ctx sdk.Context) []types.ChainConfig                { return a.k.GetAllChainConfigs(ctx) }
func (a *keeperAdapter) GetBridgeValidator(ctx sdk.Context, addr string) (types.BridgeValidator, bool) {
	return a.k.GetBridgeValidator(ctx, addr)
}
func (a *keeperAdapter) SetBridgeValidator(ctx sdk.Context, v types.BridgeValidator) error     { return a.k.SetBridgeValidator(ctx, v) }
func (a *keeperAdapter) GetAllBridgeValidators(ctx sdk.Context) []types.BridgeValidator        { return a.k.GetAllBridgeValidators(ctx) }
func (a *keeperAdapter) GetActiveValidatorsForChain(ctx sdk.Context, id string) []types.BridgeValidator {
	return a.k.GetActiveValidatorsForChain(ctx, id)
}
func (a *keeperAdapter) GetOperation(ctx sdk.Context, id string) (types.BridgeOperation, bool) { return a.k.GetOperation(ctx, id) }
func (a *keeperAdapter) SetOperation(ctx sdk.Context, op types.BridgeOperation) error          { return a.k.SetOperation(ctx, op) }
func (a *keeperAdapter) GetAllOperations(ctx sdk.Context) []types.BridgeOperation              { return a.k.GetAllOperations(ctx) }
func (a *keeperAdapter) NextOperationID(ctx sdk.Context) string                                { return a.k.NextOperationID(ctx) }
func (a *keeperAdapter) GetLockedAmount(ctx sdk.Context, chain, asset string) types.LockedAmount {
	return a.k.GetLockedAmount(ctx, chain, asset)
}
func (a *keeperAdapter) SetLockedAmount(ctx sdk.Context, la types.LockedAmount) error          { return a.k.SetLockedAmount(ctx, la) }
func (a *keeperAdapter) GetAllLockedAmounts(ctx sdk.Context) []types.LockedAmount              { return a.k.GetAllLockedAmounts(ctx) }
func (a *keeperAdapter) GetCircuitBreaker(ctx sdk.Context, chain string) types.CircuitBreakerState {
	return a.k.GetCircuitBreaker(ctx, chain)
}
func (a *keeperAdapter) SetCircuitBreaker(ctx sdk.Context, cb types.CircuitBreakerState) error { return a.k.SetCircuitBreaker(ctx, cb) }
func (a *keeperAdapter) GetAllCircuitBreakers(ctx sdk.Context) []types.CircuitBreakerState     { return a.k.GetAllCircuitBreakers(ctx) }
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState)                    { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState                     { return a.k.ExportGenesis(ctx) }

// pqcKeeperBridge adapts the pqcmod.PQCKeeper interface to what bridge keeper
// needs from pqckeeper.Keeper. Since bridge's keeper.go imports pqckeeper.Keeper
// directly, we need to provide a concrete pqckeeper.Keeper. We achieve this by
// using the pqc.keeperAdapter's internal concrete keeper.
//
// This function extracts the concrete pqckeeper.Keeper from the PQCKeeper
// interface provided by the factory.
func extractPQCKeeper(pqcKeeper pqcmod.PQCKeeper) interface{} {
	return pqcKeeper
}

// RealNewBridgeKeeper creates the real bridge keeper.
func RealNewBridgeKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, pqcKeeper pqcmod.PQCKeeper, burnKeeper burnmod.BurnKeeper, logger log.Logger) BridgeKeeper {
	// The bridge keeper needs a concrete pqckeeper.Keeper, but we receive a
	// pqcmod.PQCKeeper interface. In the proprietary build, this is a
	// *pqc.keeperAdapter wrapping a pqckeeper.Keeper. We use a type-switch
	// approach: if it's the adapter, extract the concrete keeper.
	// For this to work, pqc.register.go exports a helper.
	concreteKeeper := pqcmod.ExtractConcreteKeeper(pqcKeeper)
	k := keeper.NewKeeper(cdc, storeKey, concreteKeeper, burnKeeper, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real bridge AppModule.
func RealNewAppModule(k BridgeKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("BridgeKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
