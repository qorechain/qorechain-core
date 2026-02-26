package rlconsensus

import (
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// RLConsensusKeeper defines the interface for the RL consensus module keeper.
// Both the proprietary and stub implementations satisfy this interface.
type RLConsensusKeeper interface {
	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error
	GetAgentStatus(ctx sdk.Context) types.AgentStatus
	GetLatestObservation(ctx sdk.Context) (*types.Observation, error)
	GetLatestReward(ctx sdk.Context) (*types.Reward, error)
	GetPolicyWeights(ctx sdk.Context) (*types.PolicyWeights, error)

	// RLConsensusParamsProvider interface (drop-in for StaticRLProvider in x/vm/precompiles)
	GetCurrentBlockTime(ctx sdk.Context) time.Duration
	GetCurrentBaseGasPrice(ctx sdk.Context) math.LegacyDec
	GetValidatorSetSize(ctx sdk.Context) uint64
	GetCurrentEpoch(ctx sdk.Context) uint64
	IsRLActive(ctx sdk.Context) bool

	// v1.3.0 RDK integration — advisory rollup configuration
	SuggestRollupProfile(ctx sdk.Context, useCase string) (string, error)
	OptimizeRollupGas(ctx sdk.Context, metrics map[string]uint64) (uint64, error)

	// ABCI hooks
	BeginBlock(ctx sdk.Context) error
	EndBlock(ctx sdk.Context) error

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState

	Logger() log.Logger
}

// TokenomicsKeeper defines the interface for querying xQORE balances.
type TokenomicsKeeper interface {
	GetXQOREBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int
}

// NilTokenomicsKeeper is a no-op implementation of TokenomicsKeeper.
type NilTokenomicsKeeper struct{}

// GetXQOREBalance always returns zero for the nil implementation.
func (NilTokenomicsKeeper) GetXQOREBalance(_ sdk.Context, _ sdk.AccAddress) math.Int {
	return math.ZeroInt()
}
