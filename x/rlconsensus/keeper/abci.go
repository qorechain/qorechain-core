//go:build proprietary

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// BeginBlock collects observations every ObservationInterval blocks.
func (k *Keeper) BeginBlock(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	if !params.Enabled || params.AgentMode == types.AgentModePaused {
		return nil
	}

	height := ctx.BlockHeight()
	if height <= 0 || params.ObservationInterval == 0 {
		return nil
	}

	if uint64(height)%params.ObservationInterval != 0 {
		return nil
	}

	obs, err := k.CollectObservation(ctx)
	if err != nil {
		k.logger.Error("failed to collect observation", "height", height, "error", err)
		return nil // don't halt the chain
	}

	if err := k.SetObservation(ctx, obs); err != nil {
		k.logger.Error("failed to store observation", "height", height, "error", err)
		return nil
	}

	// Emit observation event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeObservationCollected,
			sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", height)),
		),
	)

	return nil
}

// EndBlock computes reward, runs inference, and applies actions every
// ObservationInterval blocks.
func (k *Keeper) EndBlock(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	if !params.Enabled || params.AgentMode == types.AgentModePaused {
		return nil
	}

	height := ctx.BlockHeight()
	if height <= 0 || params.ObservationInterval == 0 {
		return nil
	}

	if uint64(height)%params.ObservationInterval != 0 {
		return nil
	}

	// If circuit breaker is active, track recovery blocks
	status := k.GetAgentStatus(ctx)
	if status.CircuitBreakerActive {
		status.BlocksSinceRevert++

		// Auto-recover after 2x the circuit breaker window
		recoveryBlocks := int64(params.CircuitBreakerWindow) * 2
		if status.BlocksSinceRevert >= recoveryBlocks {
			status.CircuitBreakerActive = false
			status.BlocksSinceRevert = 0
			k.logger.Info("circuit breaker recovered", "height", height)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeCircuitBreakerRecovered,
					sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", height)),
				),
			)
		}

		if err := k.SetAgentStatus(ctx, status); err != nil {
			k.logger.Error("failed to update agent status", "error", err)
		}

		if status.CircuitBreakerActive {
			return nil // still in recovery, skip agent actions
		}
	}

	// Get current and previous observations
	curr, err := k.GetLatestObservation(ctx)
	if err != nil || curr == nil {
		return nil
	}

	prevHeight := height - int64(params.ObservationInterval)
	prev, _ := k.GetObservation(ctx, prevHeight)
	if prev == nil {
		return nil // need two observations to compute reward
	}

	// Compute reward
	reward, err := k.ComputeReward(ctx, prev, curr)
	if err != nil {
		k.logger.Error("failed to compute reward", "error", err)
		return nil
	}
	if err := k.SetReward(ctx, reward); err != nil {
		k.logger.Error("failed to store reward", "error", err)
	}

	// Emit reward event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewardComputed,
			sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", height)),
			sdk.NewAttribute(types.AttributeKeyReward, reward.TotalReward),
		),
	)

	// Run agent inference (if we have policy weights)
	if k.agent == nil {
		pw, pwErr := k.GetPolicyWeights(ctx)
		if pwErr != nil || pw == nil {
			return nil // no policy loaded yet
		}
		k.agent = NewPPOAgent(pw)
	}

	action, err := k.agent.Infer(curr)
	if err != nil {
		k.logger.Error("agent inference failed", "error", err)
		return nil
	}

	// Apply actions (shadow mode just logs)
	if err := k.ApplyActions(ctx, action); err != nil {
		k.logger.Error("failed to apply actions", "error", err)
	}

	// Circuit breaker check
	triggered, cbErr := k.CheckCircuitBreaker(ctx)
	if cbErr != nil {
		k.logger.Error("circuit breaker check failed", "error", cbErr)
	}
	if triggered {
		if err := k.TriggerCircuitBreaker(ctx); err != nil {
			k.logger.Error("failed to trigger circuit breaker", "error", err)
		}
	}

	// Update agent status
	status = k.GetAgentStatus(ctx) // re-read in case circuit breaker modified it
	status.TotalSteps++
	status.LastObservationAt = height
	status.LastActionAt = height
	if err := k.SetAgentStatus(ctx, status); err != nil {
		k.logger.Error("failed to update agent status", "error", err)
	}

	return nil
}
