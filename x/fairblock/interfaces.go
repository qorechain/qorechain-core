package fairblock

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/fairblock/types"
)

// FairBlockKeeper is the interface for the x/fairblock module keeper.
type FairBlockKeeper interface {
	Logger() log.Logger

	GetConfig(ctx sdk.Context) types.FairBlockConfig
	SetConfig(ctx sdk.Context, config types.FairBlockConfig) error
	IsEnabled(ctx sdk.Context) bool

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
