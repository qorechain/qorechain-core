package inflation

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/inflation/types"
)

// InflationKeeper is the interface for the x/inflation module's keeper.
type InflationKeeper interface {
	Logger() log.Logger

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error

	GetCurrentInflationRate(ctx sdk.Context) math.LegacyDec
	GetEpochInfo(ctx sdk.Context) types.EpochInfo
	MintEpochEmission(ctx sdk.Context) error

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
