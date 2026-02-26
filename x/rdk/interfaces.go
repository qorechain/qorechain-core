package rdk

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// RDKKeeper is the interface for the x/rdk module keeper.
type RDKKeeper interface {
	Logger() log.Logger

	// Rollup Lifecycle
	CreateRollup(ctx sdk.Context, config types.RollupConfig) (*types.RollupConfig, error)
	PauseRollup(ctx sdk.Context, rollupID string, reason string) error
	ResumeRollup(ctx sdk.Context, rollupID string) error
	StopRollup(ctx sdk.Context, rollupID string) error
	GetRollup(ctx sdk.Context, rollupID string) (*types.RollupConfig, error)
	ListRollups(ctx sdk.Context) ([]*types.RollupConfig, error)
	ListRollupsByCreator(ctx sdk.Context, creator string) ([]*types.RollupConfig, error)

	// Settlement
	SubmitBatch(ctx sdk.Context, batch types.SettlementBatch) error
	ChallengeBatch(ctx sdk.Context, rollupID string, batchIndex uint64, proof []byte) error
	FinalizeBatch(ctx sdk.Context, rollupID string, batchIndex uint64) error
	GetBatch(ctx sdk.Context, rollupID string, batchIndex uint64) (*types.SettlementBatch, error)
	GetLatestBatch(ctx sdk.Context, rollupID string) (*types.SettlementBatch, error)

	// DA Routing
	SubmitDABlob(ctx sdk.Context, blob types.DABlob) (*types.DACommitment, error)
	GetDABlob(ctx sdk.Context, rollupID string, blobIndex uint64) (*types.DABlob, error)
	PruneExpiredBlobs(ctx sdk.Context) (uint64, error)

	// AI-Assisted Configuration
	SuggestProfile(ctx sdk.Context, useCase string) (*types.RollupProfile, error)
	OptimizeGasConfig(ctx sdk.Context, rollupID string) (*types.RollupGasConfig, error)

	// Params / Genesis
	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
