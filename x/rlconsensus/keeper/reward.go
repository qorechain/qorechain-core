//go:build proprietary

package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// ComputeReward computes the reward signal from two consecutive observations.
//
// Formula: R = w1*delta_throughput + w2*delta_finality + w3*delta_decentralization
//            - w4*mev - w5*failed_txs
//
// All arithmetic uses math.LegacyDec. Weights are read from module params.
func (k *Keeper) ComputeReward(ctx sdk.Context, prev, curr *types.Observation) (*types.Reward, error) {
	params := k.GetParams(ctx)
	rw := params.RewardWeights

	w1, err := math.LegacyNewDecFromStr(rw.Throughput)
	if err != nil {
		return nil, fmt.Errorf("invalid throughput weight: %w", err)
	}
	w2, err := math.LegacyNewDecFromStr(rw.Finality)
	if err != nil {
		return nil, fmt.Errorf("invalid finality weight: %w", err)
	}
	w3, err := math.LegacyNewDecFromStr(rw.Decentralization)
	if err != nil {
		return nil, fmt.Errorf("invalid decentralization weight: %w", err)
	}
	w4, err := math.LegacyNewDecFromStr(rw.MEV)
	if err != nil {
		return nil, fmt.Errorf("invalid MEV weight: %w", err)
	}
	w5, err := math.LegacyNewDecFromStr(rw.FailedTxs)
	if err != nil {
		return nil, fmt.Errorf("invalid failed_txs weight: %w", err)
	}

	// Compute deltas between current and previous observations.

	// Throughput delta: change in block utilization (higher is better)
	throughputDelta := decDelta(curr.Values[types.ObsBlockUtilization], prev.Values[types.ObsBlockUtilization])

	// Finality delta: improvement in precommit ratio (higher is better)
	finalityDelta := decDelta(curr.Values[types.ObsPrecommitRatio], prev.Values[types.ObsPrecommitRatio])

	// Decentralization delta: reduction in Gini coefficient (lower Gini is better, so negate)
	giniDelta := decDelta(curr.Values[types.ObsValidatorGini], prev.Values[types.ObsValidatorGini])
	decentralizationDelta := giniDelta.Neg() // Decreasing Gini = positive reward

	// MEV estimate: current value (lower is better, penalized)
	mevEstimate := parseDec(curr.Values[types.ObsMEVEstimate])

	// Failed tx ratio: current value (lower is better, penalized)
	failedTxRatio := parseDec(curr.Values[types.ObsFailedTxRatio])

	// Total reward = w1*throughput + w2*finality + w3*decentralization - w4*mev - w5*failed
	totalReward := w1.Mul(throughputDelta).
		Add(w2.Mul(finalityDelta)).
		Add(w3.Mul(decentralizationDelta)).
		Sub(w4.Mul(mevEstimate)).
		Sub(w5.Mul(failedTxRatio))

	reward := &types.Reward{
		Height:                curr.Height,
		TotalReward:           totalReward.String(),
		ThroughputDelta:       throughputDelta.String(),
		FinalityDelta:         finalityDelta.String(),
		DecentralizationDelta: decentralizationDelta.String(),
		MEVEstimate:           mevEstimate.String(),
		FailedTxRatio:         failedTxRatio.String(),
	}

	return reward, nil
}

// decDelta computes currStr - prevStr as LegacyDec.
func decDelta(currStr, prevStr string) math.LegacyDec {
	curr := parseDec(currStr)
	prev := parseDec(prevStr)
	return curr.Sub(prev)
}

// parseDec parses a LegacyDec string, returning zero on error.
func parseDec(s string) math.LegacyDec {
	if s == "" {
		return math.LegacyZeroDec()
	}
	d, err := math.LegacyNewDecFromStr(s)
	if err != nil {
		return math.LegacyZeroDec()
	}
	return d
}
