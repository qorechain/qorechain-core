package babylon

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/babylon/types"
)

// BabylonKeeper is the interface for the x/babylon module keeper.
type BabylonKeeper interface {
	Logger() log.Logger

	// Config
	GetConfig(ctx sdk.Context) types.BTCRestakingConfig
	SetConfig(ctx sdk.Context, config types.BTCRestakingConfig) error
	IsEnabled(ctx sdk.Context) bool

	// Staking positions
	GetStakingPosition(ctx sdk.Context, id string) (types.BTCStakingPosition, bool)
	SetStakingPosition(ctx sdk.Context, pos types.BTCStakingPosition) error
	GetAllPositions(ctx sdk.Context) []types.BTCStakingPosition

	// Checkpoints
	GetCheckpoint(ctx sdk.Context, epoch uint64) (types.BTCCheckpoint, bool)
	SetCheckpoint(ctx sdk.Context, cp types.BTCCheckpoint) error

	// Epochs
	GetCurrentEpoch(ctx sdk.Context) uint64
	GetEpochSnapshot(ctx sdk.Context, epoch uint64) (types.BabylonEpochSnapshot, bool)

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
