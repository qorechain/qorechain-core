//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// QueryAgentStatus returns the current RL agent status.
func (k *Keeper) QueryAgentStatus(ctx sdk.Context) types.AgentStatus {
	return k.GetAgentStatus(ctx)
}

// QueryLatestObservation returns the most recent observation vector.
func (k *Keeper) QueryLatestObservation(ctx sdk.Context) (*types.Observation, error) {
	return k.GetLatestObservation(ctx)
}

// QueryLatestReward returns the most recent reward signal.
func (k *Keeper) QueryLatestReward(ctx sdk.Context) (*types.Reward, error) {
	return k.GetLatestReward(ctx)
}

// QueryParams returns the current module parameters.
func (k *Keeper) QueryParams(ctx sdk.Context) types.Params {
	return k.GetParams(ctx)
}

// QueryPolicyWeights returns the current policy network weights.
func (k *Keeper) QueryPolicyWeights(ctx sdk.Context) (*types.PolicyWeights, error) {
	return k.GetPolicyWeights(ctx)
}

// QueryAppliedParams returns the currently applied consensus parameters.
func (k *Keeper) QueryAppliedParams(ctx sdk.Context) AppliedConsensusParams {
	return k.GetAppliedParams(ctx)
}
