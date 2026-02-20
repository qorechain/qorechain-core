//go:build proprietary

package ai

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/ai/keeper"
	"github.com/qorechain/qorechain-core/x/ai/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the AIKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Engine() types.AIEngine                { return a.k.Engine() }
func (a *keeperAdapter) Logger() log.Logger                     { return a.k.Logger() }
func (a *keeperAdapter) GetConfig(ctx sdk.Context) types.AIConfig { return a.k.GetConfig(ctx) }
func (a *keeperAdapter) SetConfig(ctx sdk.Context, cfg types.AIConfig) error { return a.k.SetConfig(ctx, cfg) }
func (a *keeperAdapter) GetStats(ctx sdk.Context) types.AIStats { return a.k.GetStats(ctx) }
func (a *keeperAdapter) SetStats(ctx sdk.Context, s types.AIStats) { a.k.SetStats(ctx, s) }
func (a *keeperAdapter) IncrementTxsRouted(ctx sdk.Context)     { a.k.IncrementTxsRouted(ctx) }
func (a *keeperAdapter) IncrementAnomaliesDetected(ctx sdk.Context) { a.k.IncrementAnomaliesDetected(ctx) }
func (a *keeperAdapter) FlagTransaction(ctx sdk.Context, f types.FlaggedTx) { a.k.FlagTransaction(ctx, f) }
func (a *keeperAdapter) AnalyzeTransaction(ctx sdk.Context, tx types.TransactionInfo, history []types.TransactionInfo) (*types.AnomalyResult, error) {
	return a.k.AnalyzeTransaction(ctx, tx, history)
}
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState  { return a.k.ExportGenesis(ctx) }

// RealNewAIKeeper creates the real AI keeper with the heuristic engine.
func RealNewAIKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) AIKeeper {
	engine := keeper.NewHeuristicEngine(10) // max 10 tx/min per sender
	k := keeper.NewKeeper(cdc, storeKey, engine, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real AI AppModule.
func RealNewAppModule(k AIKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("AIKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}

// RealNewAIAnomalyDecorator creates the real AI ante decorator.
func RealNewAIAnomalyDecorator(k AIKeeper) sdk.AnteDecorator {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("AIKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAIAnomalyDecorator(adapter.k)
}
