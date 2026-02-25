//go:build proprietary

package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// CircuitBreakerState tracks recent block times to detect chain instability.
type CircuitBreakerState struct {
	RecentBlockTimes  []int64 `json:"recent_block_times"`   // last N block times in ms
	TargetBlockTimeMs int64   `json:"target_block_time_ms"`
	IsTriggered       bool    `json:"is_triggered"`
}

// CheckCircuitBreaker evaluates whether the circuit breaker should trigger.
// It counts how many of the last CircuitBreakerWindow blocks had block time
// within 2x of target. If the fraction of "healthy" blocks falls below the
// threshold, the circuit breaker triggers.
func (k *Keeper) CheckCircuitBreaker(ctx sdk.Context) (triggered bool, err error) {
	params := k.GetParams(ctx)
	state := k.GetCircuitBreakerState(ctx)

	// Update recent block times
	applied := k.GetAppliedParams(ctx)
	targetMs := applied.BlockTimeMs
	if targetMs <= 0 {
		targetMs = params.DefaultBlockTimeMs
	}
	state.TargetBlockTimeMs = targetMs

	// Use block header time as a proxy for actual block time.
	// We record the current block time; actual delta is computed from the list.
	currentTimeMs := ctx.BlockHeader().Time.UnixMilli()
	state.RecentBlockTimes = append(state.RecentBlockTimes, currentTimeMs)

	// Keep only the last CircuitBreakerWindow entries
	window := int(params.CircuitBreakerWindow)
	if len(state.RecentBlockTimes) > window {
		state.RecentBlockTimes = state.RecentBlockTimes[len(state.RecentBlockTimes)-window:]
	}

	// Need at least 2 entries to compute deltas
	if len(state.RecentBlockTimes) < 2 {
		if err := k.SetCircuitBreakerState(ctx, state); err != nil {
			return false, err
		}
		return false, nil
	}

	// Count how many block time deltas are "healthy" (within 2x of target)
	healthyCount := 0
	totalDeltas := len(state.RecentBlockTimes) - 1
	maxAllowed := targetMs * 2

	for i := 1; i < len(state.RecentBlockTimes); i++ {
		delta := state.RecentBlockTimes[i] - state.RecentBlockTimes[i-1]
		if delta < 0 {
			delta = -delta
		}
		if delta <= maxAllowed && delta > 0 {
			healthyCount++
		}
	}

	// Parse threshold
	threshold, threshErr := math.LegacyNewDecFromStr(params.CircuitBreakerThreshold)
	if threshErr != nil {
		return false, fmt.Errorf("invalid circuit breaker threshold: %w", threshErr)
	}

	// Healthy fraction
	if totalDeltas == 0 {
		if err := k.SetCircuitBreakerState(ctx, state); err != nil {
			return false, err
		}
		return false, nil
	}

	healthyFraction := math.LegacyNewDec(int64(healthyCount)).Quo(math.LegacyNewDec(int64(totalDeltas)))

	// If healthy fraction falls below threshold, trigger
	if healthyFraction.LT(threshold) {
		state.IsTriggered = true
		if err := k.SetCircuitBreakerState(ctx, state); err != nil {
			return true, err
		}
		return true, nil
	}

	// If previously triggered and now recovered, clear
	if state.IsTriggered && healthyFraction.GTE(threshold) {
		state.IsTriggered = false
	}

	if err := k.SetCircuitBreakerState(ctx, state); err != nil {
		return false, err
	}
	return false, nil
}

// TriggerCircuitBreaker reverts all RL-tuned params to defaults and pauses the agent.
func (k *Keeper) TriggerCircuitBreaker(ctx sdk.Context) error {
	params := k.GetParams(ctx)

	// Revert applied params to defaults
	defaultApplied := AppliedConsensusParams{
		BlockTimeMs:    params.DefaultBlockTimeMs,
		GasLimit:       0, // will use chain default
		GasPriceFloor:  params.DefaultBaseGasPrice,
		PoolWeightRPoS: "0.400000000000000000",
		PoolWeightDPoS: "0.350000000000000000",
	}
	if err := k.SetAppliedParams(ctx, defaultApplied); err != nil {
		return fmt.Errorf("failed to revert applied params: %w", err)
	}

	// Update agent status: mark circuit breaker active, reset blocks since revert
	status := k.GetAgentStatus(ctx)
	status.CircuitBreakerActive = true
	status.BlocksSinceRevert = 0
	if err := k.SetAgentStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	// Clear the in-memory agent to force reload on next cycle
	k.agent = nil

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCircuitBreakerTriggered,
			sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	k.logger.Info("circuit breaker triggered, reverted to default parameters",
		"height", ctx.BlockHeight())

	return nil
}
