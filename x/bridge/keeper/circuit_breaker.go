//go:build proprietary

package keeper

import (
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// BridgeCircuitBreaker implements circuit breaker logic for bridge transfers.
// Per whitepaper:
// - Single transfer limit enforcement
// - Daily aggregate limit tracking
// - Manual pause capability for emergency response
// - Automatic trip on anomalous activity
type BridgeCircuitBreaker struct {
	keeper Keeper
}

// NewBridgeCircuitBreaker creates a new circuit breaker.
func NewBridgeCircuitBreaker(k Keeper) *BridgeCircuitBreaker {
	return &BridgeCircuitBreaker{keeper: k}
}

// CheckTransfer validates a transfer against all circuit breaker conditions.
func (cb *BridgeCircuitBreaker) CheckTransfer(ctx sdk.Context, chain string, amount sdkmath.Int) error {
	return cb.keeper.CheckCircuitBreakerLimits(ctx, chain, amount)
}

// PauseBridge pauses the bridge for a specific chain.
func (cb *BridgeCircuitBreaker) PauseBridge(ctx sdk.Context, chain string, reason string) error {
	state := cb.keeper.GetCircuitBreaker(ctx, chain)
	state.Paused = true
	state.PausedReason = reason
	now := ctx.BlockTime()
	state.PausedAt = &now

	if err := cb.keeper.SetCircuitBreaker(ctx, state); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCircuitBreakerTrip,
		sdk.NewAttribute(types.AttributeKeyChain, chain),
		sdk.NewAttribute(types.AttributeKeyStatus, "paused"),
		sdk.NewAttribute("reason", reason),
	))

	cb.keeper.Logger().Warn("bridge circuit breaker activated",
		"chain", chain,
		"reason", reason,
	)

	return nil
}

// ResumeBridge resumes a paused bridge.
func (cb *BridgeCircuitBreaker) ResumeBridge(ctx sdk.Context, chain string) error {
	state := cb.keeper.GetCircuitBreaker(ctx, chain)
	state.Paused = false
	state.PausedReason = ""
	state.PausedAt = nil

	if err := cb.keeper.SetCircuitBreaker(ctx, state); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCircuitBreakerTrip,
		sdk.NewAttribute(types.AttributeKeyChain, chain),
		sdk.NewAttribute(types.AttributeKeyStatus, "resumed"),
	))

	cb.keeper.Logger().Info("bridge circuit breaker deactivated", "chain", chain)

	return nil
}

// UpdateLimits updates circuit breaker limits for a chain.
func (cb *BridgeCircuitBreaker) UpdateLimits(ctx sdk.Context, chain string, maxSingle, dailyLimit sdkmath.Int) error {
	state := cb.keeper.GetCircuitBreaker(ctx, chain)
	state.MaxSingleTransfer = maxSingle.String()
	state.DailyLimit = dailyLimit.String()
	return cb.keeper.SetCircuitBreaker(ctx, state)
}

// ResetDailyCounters resets all chain daily counters. Called periodically.
func (cb *BridgeCircuitBreaker) ResetDailyCounters(ctx sdk.Context) {
	chains := cb.keeper.GetAllChainConfigs(ctx)
	for _, chain := range chains {
		state := cb.keeper.GetCircuitBreaker(ctx, chain.ChainID)
		state.CurrentDaily = "0"
		state.LastResetHeight = ctx.BlockHeight()
		_ = cb.keeper.SetCircuitBreaker(ctx, state)
	}
}

// CheckAndTripOnAnomaly checks if bridge activity is anomalous and trips the breaker.
// This integrates with the AI fraud detection system.
func (cb *BridgeCircuitBreaker) CheckAndTripOnAnomaly(ctx sdk.Context, chain string, recentOpsCount int, recentVolume sdkmath.Int) {
	// Heuristic: if more than 50 operations in a short window, investigate
	if recentOpsCount > 50 {
		_ = cb.PauseBridge(ctx, chain, "anomalous operation volume detected")
		return
	}

	// Heuristic: if volume exceeds 5x daily limit in short window
	state := cb.keeper.GetCircuitBreaker(ctx, chain)
	dailyLimit, _ := types.ParseAmount(state.DailyLimit)
	if !dailyLimit.IsZero() && recentVolume.GT(dailyLimit.MulRaw(5)) {
		_ = cb.PauseBridge(ctx, chain, "anomalous volume detected: exceeds 5x daily limit")
	}
}

// ProcessChallengedOperations checks attested operations whose challenge period has ended.
func (cb *BridgeCircuitBreaker) ProcessChallengedOperations(ctx sdk.Context) {
	ops := cb.keeper.GetAllOperations(ctx)
	now := ctx.BlockTime()

	for _, op := range ops {
		if op.Status != types.OpStatusAttested {
			continue
		}
		if op.ChallengeEndTime == nil {
			continue
		}
		if now.Before(*op.ChallengeEndTime) {
			continue
		}

		// Challenge period expired — execute the operation
		if err := cb.keeper.ExecuteOperation(ctx, &op); err != nil {
			cb.keeper.Logger().Error("failed to execute challenged operation",
				"operation_id", op.ID,
				"error", err,
			)
			op.Status = types.OpStatusFailed
		}
		_ = cb.keeper.SetOperation(ctx, op)
	}
}

// GetChallengeEndTime calculates when a challenge period ends for a given time.
func GetChallengeEndTime(now time.Time, challengePeriodSecs int64) time.Time {
	return now.Add(time.Duration(challengePeriodSecs) * time.Second)
}
