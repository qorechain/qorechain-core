package burn

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/burn/types"
)

// BurnKeeper is the interface for the x/burn module's keeper.
type BurnKeeper interface {
	Logger() log.Logger

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error

	// Core burn operations
	BurnFromSource(ctx sdk.Context, source types.BurnSource, amount math.Int, txHash string) error
	GetTotalBurned(ctx sdk.Context) math.Int
	GetBurnStats(ctx sdk.Context) types.BurnStats
	GetBurnRecords(ctx sdk.Context, limit int) []types.BurnRecord

	// Fee distribution (EndBlocker)
	DistributeFees(ctx sdk.Context) error

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
