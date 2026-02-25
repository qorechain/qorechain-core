//go:build proprietary

package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// AppliedConsensusParams holds the RL-applied parameter values.
type AppliedConsensusParams struct {
	BlockTimeMs    int64  `json:"block_time_ms"`
	GasLimit       int64  `json:"gas_limit"`
	GasPriceFloor  string `json:"gas_price_floor"`   // LegacyDec string
	PoolWeightRPoS string `json:"pool_weight_rpos"`   // LegacyDec string
	PoolWeightDPoS string `json:"pool_weight_dpos"`   // LegacyDec string
}

// DefaultAppliedParams returns the default applied consensus parameters.
func DefaultAppliedParams() AppliedConsensusParams {
	return AppliedConsensusParams{
		BlockTimeMs:    types.DefaultBlockTimeMs,
		GasLimit:       0,
		GasPriceFloor:  types.DefaultBaseGasPrice,
		PoolWeightRPoS: "0.400000000000000000",
		PoolWeightDPoS: "0.350000000000000000",
	}
}

// ApplyActions applies the agent's action vector to consensus parameters.
// In shadow mode, only logs the proposed changes. In conservative/autonomous
// modes, applies changes with clamping to the configured max change bounds.
func (k *Keeper) ApplyActions(ctx sdk.Context, action *types.Action) error {
	params := k.GetParams(ctx)
	maxChange := params.MaxChangeForMode()
	current := k.GetAppliedParams(ctx)

	// In shadow or paused mode, just log and return
	if params.AgentMode == types.AgentModeShadow || params.AgentMode == types.AgentModePaused {
		k.logger.Info("shadow mode: logging proposed action without applying",
			"height", ctx.BlockHeight(),
			"block_time_delta", action.Values[types.ActBlockTimeDelta],
			"gas_price_delta", action.Values[types.ActGasPriceDelta],
			"val_set_delta", action.Values[types.ActValidatorSetSizeDelta],
			"pool_pqc_delta", action.Values[types.ActPoolWeightPQCDelta],
			"pool_dpos_delta", action.Values[types.ActPoolWeightDPoSDelta],
		)
		return nil
	}

	// Circuit breaker active: do not apply
	status := k.GetAgentStatus(ctx)
	if status.CircuitBreakerActive {
		k.logger.Info("circuit breaker active: skipping action application")
		return nil
	}

	// Apply each action dimension with clamping

	// 1. Block time delta (ms)
	blockTimeDelta := parseDec(action.Values[types.ActBlockTimeDelta])
	blockTimeDelta = clampDelta(blockTimeDelta, maxChange)
	newBlockTimeMs := current.BlockTimeMs + blockTimeDelta.TruncateInt64()
	if newBlockTimeMs < 1000 { // minimum 1 second
		newBlockTimeMs = 1000
	}
	if newBlockTimeMs > 30000 { // maximum 30 seconds
		newBlockTimeMs = 30000
	}
	current.BlockTimeMs = newBlockTimeMs

	// 2. Gas price floor delta
	gasPriceDelta := parseDec(action.Values[types.ActGasPriceDelta])
	gasPriceDelta = clampDelta(gasPriceDelta, maxChange)
	currentGasPrice := parseDec(current.GasPriceFloor)
	newGasPrice := currentGasPrice.Add(gasPriceDelta)
	if newGasPrice.IsNegative() {
		newGasPrice = math.LegacyOneDec() // minimum gas price of 1
	}
	current.GasPriceFloor = newGasPrice.String()

	// 3. Validator set size delta (action index 2 - logged but not applied directly)
	// Validator set size is a governance parameter; RL can recommend but not enforce.

	// 4. Pool weight RPoS delta
	rposDelta := parseDec(action.Values[types.ActPoolWeightPQCDelta])
	rposDelta = clampDelta(rposDelta, maxChange)
	currentRPoS := parseDec(current.PoolWeightRPoS)
	newRPoS := currentRPoS.Add(rposDelta)
	newRPoS = clampWeight(newRPoS)
	current.PoolWeightRPoS = newRPoS.String()

	// 5. Pool weight DPoS delta
	dposDelta := parseDec(action.Values[types.ActPoolWeightDPoSDelta])
	dposDelta = clampDelta(dposDelta, maxChange)
	currentDPoS := parseDec(current.PoolWeightDPoS)
	newDPoS := currentDPoS.Add(dposDelta)
	newDPoS = clampWeight(newDPoS)
	current.PoolWeightDPoS = newDPoS.String()

	// Persist
	if err := k.SetAppliedParams(ctx, current); err != nil {
		return fmt.Errorf("failed to persist applied params: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeActionApplied,
			sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute(types.AttributeKeyAgentMode, params.AgentMode.String()),
		),
	)

	k.logger.Info("applied RL action",
		"height", ctx.BlockHeight(),
		"mode", params.AgentMode.String(),
		"block_time_ms", current.BlockTimeMs,
		"gas_price_floor", current.GasPriceFloor,
		"pool_rpos", current.PoolWeightRPoS,
		"pool_dpos", current.PoolWeightDPoS,
	)

	return nil
}

// clampDelta clamps a delta value to [-maxChange, +maxChange].
func clampDelta(delta, maxChange math.LegacyDec) math.LegacyDec {
	if maxChange.IsZero() {
		return math.LegacyZeroDec()
	}
	if delta.GT(maxChange) {
		return maxChange
	}
	if delta.LT(maxChange.Neg()) {
		return maxChange.Neg()
	}
	return delta
}

// clampWeight clamps a pool weight to [0.05, 0.80].
func clampWeight(w math.LegacyDec) math.LegacyDec {
	minWeight := math.LegacyMustNewDecFromStr("0.05")
	maxWeight := math.LegacyMustNewDecFromStr("0.80")
	if w.LT(minWeight) {
		return minWeight
	}
	if w.GT(maxWeight) {
		return maxWeight
	}
	return w
}
