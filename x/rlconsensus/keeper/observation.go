//go:build proprietary

package keeper

import (
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// CollectObservation gathers the 25-dimension observation vector from on-chain state.
// Missing or unavailable data dimensions are set to safe defaults (zero).
func (k *Keeper) CollectObservation(ctx sdk.Context) (*types.Observation, error) {
	height := ctx.BlockHeight()
	obs := &types.Observation{
		Height: height,
	}

	// Initialize all values to "0" as safe default
	for i := 0; i < types.ObservationDimensions; i++ {
		obs.Values[i] = "0"
	}

	header := ctx.BlockHeader()

	// --- Block metrics ---

	gasUsed := ctx.BlockGasMeter().GasConsumed()
	gasLimit := ctx.BlockGasMeter().Limit()
	if gasLimit == 0 {
		gasLimit = 1 // avoid division by zero
	}

	// ObsBlockUtilization: gas used / gas limit
	utilization := sdkmath.LegacyNewDec(int64(gasUsed)).Quo(sdkmath.LegacyNewDec(int64(gasLimit)))
	obs.Values[types.ObsBlockUtilization] = utilization.String()

	// ObsTxCount: not directly available from block header in this version;
	// use zero as placeholder (full implementation would count DeliverTx calls).
	obs.Values[types.ObsTxCount] = "0"

	// ObsBlockTime: time since previous block (ms) — use header time
	blockTimeMs := int64(0)
	if !header.Time.IsZero() {
		blockTimeMs = header.Time.UnixMilli()
	}
	obs.Values[types.ObsBlockTime] = sdkmath.LegacyNewDec(blockTimeMs).String()

	// ObsBlockTimeDelta: block time - target block time (ms)
	params := k.GetParams(ctx)
	targetMs := params.DefaultBlockTimeMs
	applied := k.GetAppliedParams(ctx)
	if applied.BlockTimeMs > 0 {
		targetMs = applied.BlockTimeMs
	}
	// We estimate actual block time from the applied params since we can't
	// easily get the previous block time from a single ctx. Use 0 as delta placeholder.
	obs.Values[types.ObsBlockTimeDelta] = "0"

	// ObsGasPrice50th, ObsGasPrice95th: placeholders (requires mempool inspection)
	obs.Values[types.ObsGasPrice50th] = applied.GasPriceFloor
	obs.Values[types.ObsGasPrice95th] = applied.GasPriceFloor

	// ObsMempoolSize, ObsMempoolBytes: not available from sdk.Context
	// Left as "0"

	// --- Validator metrics ---

	if k.reputationReader != nil {
		allReps := k.reputationReader.GetAllValidatorReputations(ctx)
		valCount := len(allReps)
		obs.Values[types.ObsValidatorCount] = sdkmath.LegacyNewDec(int64(valCount)).String()

		if valCount > 0 {
			// Compute mean and stddev of reputation scores
			var sumScore float64
			scores := make([]float64, valCount)
			for i, rep := range allReps {
				scores[i] = rep.CompositeScore
				sumScore += rep.CompositeScore
			}
			mean := sumScore / float64(valCount)
			obs.Values[types.ObsReputationMean] = float64ToDec(mean).String()

			var sumSqDiff float64
			for _, s := range scores {
				diff := s - mean
				sumSqDiff += diff * diff
			}
			stddev := math.Sqrt(sumSqDiff / float64(valCount))
			obs.Values[types.ObsReputationStdDev] = float64ToDec(stddev).String()

			// Compute Gini coefficient for validator power distribution
			gini := computeGini(scores)
			obs.Values[types.ObsValidatorGini] = float64ToDec(gini).String()
		}
	}

	// --- AI anomaly stats ---

	if k.aiReader != nil {
		anomalyCount := k.aiReader.GetAnomalyCount(ctx)
		obs.Values[types.ObsMEVEstimate] = sdkmath.LegacyNewDec(int64(anomalyCount)).String()
	}

	// --- Fee market ---

	if k.feeMarketReader != nil {
		baseFee := k.feeMarketReader.GetBaseFee(ctx)
		obs.Values[types.ObsGasPrice50th] = baseFee
	}

	// --- Current applied params (for the agent to observe its own output) ---

	obs.Values[types.ObsAvgGasPerTx] = applied.GasPriceFloor

	// ObsPrecommitRatio: default 1.0 (healthy chain)
	obs.Values[types.ObsPrecommitRatio] = "1.0"

	// ObsBondedRatio: default 0.67
	obs.Values[types.ObsBondedRatio] = "0.670000000000000000"

	// ObsInflationRate: default 0.10
	obs.Values[types.ObsInflationRate] = "0.100000000000000000"

	// ObsRewardPerValidator: placeholder
	if params.DefaultValidatorSetSize > 0 {
		obs.Values[types.ObsRewardPerValidator] = "0"
	}

	// ObsAvgTxSize: placeholder
	obs.Values[types.ObsAvgTxSize] = "256"

	// --- Historical delta (compare with params from N blocks ago) ---
	_ = targetMs // already used above

	return obs, nil
}

// float64ToDec converts a float64 to a LegacyDec using string formatting
// to maintain determinism at the boundary.
func float64ToDec(f float64) sdkmath.LegacyDec {
	d, err := sdkmath.LegacyNewDecFromStr(fmt.Sprintf("%.18f", f))
	if err != nil {
		return sdkmath.LegacyZeroDec()
	}
	return d
}

// computeGini computes the Gini coefficient for a slice of non-negative values.
// Returns 0.0 for empty or all-equal distributions.
func computeGini(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0.0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	if sum == 0 {
		return 0.0
	}

	// Gini = (2 * sum_i(i * x_i)) / (n * sum) - (n+1)/n
	// Using the relative mean absolute difference formula:
	// G = sum_i sum_j |x_i - x_j| / (2 * n * sum)
	var diffSum float64
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			diffSum += math.Abs(values[i] - values[j])
		}
	}

	gini := diffSum / (2.0 * float64(n) * sum)
	if gini < 0 {
		return 0.0
	}
	if gini > 1.0 {
		return 1.0
	}
	return gini
}
