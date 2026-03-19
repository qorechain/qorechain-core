package lightnode

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/lightnode/types"
)

// LightNodeKeeper is the interface for the x/lightnode module's keeper.
type LightNodeKeeper interface {
	Logger() log.Logger

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error

	// Registration
	RegisterLightNode(ctx sdk.Context, info types.LightNodeInfo) error
	DeregisterLightNode(ctx sdk.Context, address string) error
	GetLightNode(ctx sdk.Context, address string) (types.LightNodeInfo, bool)
	GetAllLightNodes(ctx sdk.Context) []types.LightNodeInfo
	GetLightNodesByStatus(ctx sdk.Context, status string) []types.LightNodeInfo
	GetLightNodeCount(ctx sdk.Context) uint64

	// Heartbeat
	RecordHeartbeat(ctx sdk.Context, address string, height int64) error

	// Rewards
	DistributeRewards(ctx sdk.Context, totalBlockRewards math.Int) error
	GetAccumulatedRewards(ctx sdk.Context, address string) math.Int
	ClaimRewards(ctx sdk.Context, address string) (math.Int, error)

	// Stats
	GetStats(ctx sdk.Context) types.LightNodeStats

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
