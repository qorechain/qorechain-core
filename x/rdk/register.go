//go:build proprietary

package rdk

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	burnmod "github.com/qorechain/qorechain-core/x/burn"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	"github.com/qorechain/qorechain-core/x/rdk/keeper"
	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the RDKKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) Logger() log.Logger { return a.k.Logger() }

// Rollup Lifecycle
func (a *keeperAdapter) CreateRollup(ctx sdk.Context, config types.RollupConfig) (*types.RollupConfig, error) {
	return a.k.CreateRollup(ctx, config)
}
func (a *keeperAdapter) PauseRollup(ctx sdk.Context, rollupID string, reason string) error {
	return a.k.PauseRollup(ctx, rollupID, reason)
}
func (a *keeperAdapter) ResumeRollup(ctx sdk.Context, rollupID string) error {
	return a.k.ResumeRollup(ctx, rollupID)
}
func (a *keeperAdapter) StopRollup(ctx sdk.Context, rollupID string) error {
	return a.k.StopRollup(ctx, rollupID)
}
func (a *keeperAdapter) GetRollup(ctx sdk.Context, rollupID string) (*types.RollupConfig, error) {
	return a.k.GetRollup(ctx, rollupID)
}
func (a *keeperAdapter) ListRollups(ctx sdk.Context) ([]*types.RollupConfig, error) {
	return a.k.ListRollups(ctx)
}
func (a *keeperAdapter) ListRollupsByCreator(ctx sdk.Context, creator string) ([]*types.RollupConfig, error) {
	return a.k.ListRollupsByCreator(ctx, creator)
}

// Settlement
func (a *keeperAdapter) SubmitBatch(ctx sdk.Context, batch types.SettlementBatch) error {
	return a.k.SubmitBatch(ctx, batch)
}
func (a *keeperAdapter) ChallengeBatch(ctx sdk.Context, rollupID string, batchIndex uint64, proof []byte) error {
	return a.k.ChallengeBatch(ctx, rollupID, batchIndex, proof)
}
func (a *keeperAdapter) FinalizeBatch(ctx sdk.Context, rollupID string, batchIndex uint64) error {
	return a.k.FinalizeBatch(ctx, rollupID, batchIndex)
}
func (a *keeperAdapter) GetBatch(ctx sdk.Context, rollupID string, batchIndex uint64) (*types.SettlementBatch, error) {
	return a.k.GetBatch(ctx, rollupID, batchIndex)
}
func (a *keeperAdapter) GetLatestBatch(ctx sdk.Context, rollupID string) (*types.SettlementBatch, error) {
	return a.k.GetLatestBatch(ctx, rollupID)
}

// DA Routing
func (a *keeperAdapter) SubmitDABlob(ctx sdk.Context, blob types.DABlob) (*types.DACommitment, error) {
	return a.k.SubmitDABlob(ctx, blob)
}
func (a *keeperAdapter) GetDABlob(ctx sdk.Context, rollupID string, blobIndex uint64) (*types.DABlob, error) {
	return a.k.GetDABlob(ctx, rollupID, blobIndex)
}
func (a *keeperAdapter) PruneExpiredBlobs(ctx sdk.Context) (uint64, error) {
	return a.k.PruneExpiredBlobs(ctx)
}

// AI-Assisted Configuration
func (a *keeperAdapter) SuggestProfile(ctx sdk.Context, useCase string) (*types.RollupProfile, error) {
	return a.k.SuggestProfile(ctx, useCase)
}
func (a *keeperAdapter) OptimizeGasConfig(ctx sdk.Context, rollupID string) (*types.RollupGasConfig, error) {
	return a.k.OptimizeGasConfig(ctx, rollupID)
}

// Params / Genesis
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params {
	return a.k.GetParams(ctx)
}
func (a *keeperAdapter) SetParams(ctx sdk.Context, params types.Params) error {
	return a.k.SetParams(ctx, params)
}
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	a.k.InitGenesis(ctx, gs)
}
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return a.k.ExportGenesis(ctx)
}

// RealNewRDKKeeper creates the real rdk keeper wrapped in an adapter.
func RealNewRDKKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	burnKeeper burnmod.BurnKeeper,
	multilayerKeeper multilayermod.MultilayerKeeper,
	rlKeeper rlconsensusmod.RLConsensusKeeper,
	bankKeeper bankkeeper.BaseKeeper,
	logger log.Logger,
) RDKKeeper {
	k := keeper.NewKeeper(cdc, storeKey, burnKeeper, multilayerKeeper, rlKeeper, bankKeeper, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real rdk AppModule.
func RealNewAppModule(k RDKKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("RDKKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
