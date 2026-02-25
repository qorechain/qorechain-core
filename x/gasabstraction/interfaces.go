package gasabstraction

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// GasAbstractionKeeper is the interface for the x/gasabstraction module keeper.
type GasAbstractionKeeper interface {
	Logger() log.Logger

	GetConfig(ctx sdk.Context) types.GasAbstractionConfig
	SetConfig(ctx sdk.Context, config types.GasAbstractionConfig) error
	IsEnabled(ctx sdk.Context) bool
	GetAcceptedTokens(ctx sdk.Context) []types.AcceptedFeeToken

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
