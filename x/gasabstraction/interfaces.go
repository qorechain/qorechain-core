package gasabstraction

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
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

	// Fee conversion (v3.2.2).
	// BlockConversionUsed is the native amount the float has paid out in the
	// current block; AddBlockConversionUsed adds to it.
	BlockConversionUsed(ctx sdk.Context) math.Int
	AddBlockConversionUsed(ctx sdk.Context, amount math.Int)
	// SetConversionDeps gives the keeper what the float refill needs. Every
	// copy of the keeper sees the same dependencies.
	SetConversionDeps(deps ConversionDeps)

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}

// ConversionDeps is an alias of types.ConversionDeps.
type ConversionDeps = types.ConversionDeps
