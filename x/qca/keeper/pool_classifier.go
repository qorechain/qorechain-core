//go:build proprietary

package keeper

import (
	"encoding/json"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/qca/types"
)

// ClassifyValidators classifies all active validators into the three pools.
// Call this every pool_classification_interval blocks.
func (k Keeper) ClassifyValidators(ctx sdk.Context, validators []types.ValidatorInfo) []types.PoolClassification {
	config := k.GetConfig(ctx)
	poolConfig := config.PoolConfig
	height := ctx.BlockHeight()

	if len(validators) == 0 {
		return nil
	}

	// Collect reputation scores
	repScores := make(map[string]float64, len(validators))
	for _, val := range validators {
		if val.Active {
			repScores[val.Address] = k.reputationKeeper.CalculateReputation(ctx, val.Address)
		}
	}

	// Calculate reputation percentile threshold
	var reps []float64
	for _, val := range validators {
		if val.Active {
			reps = append(reps, repScores[val.Address])
		}
	}
	sort.Float64s(reps)
	repThreshold := percentile(reps, poolConfig.RepPercentileRPoS)

	// Calculate median stake
	var stakes []uint64
	for _, val := range validators {
		if val.Active {
			stakes = append(stakes, val.Tokens)
		}
	}
	sort.Slice(stakes, func(i, j int) bool { return stakes[i] < stakes[j] })
	medianStake := median(stakes)

	// Classify each validator
	var classifications []types.PoolClassification
	for _, val := range validators {
		if !val.Active {
			continue
		}

		var pool types.PoolType

		// Priority: RPoS > DPoS > PoS
		rep := repScores[val.Address]
		if rep >= repThreshold && val.Tokens >= medianStake {
			pool = types.PoolRPoS
		} else if val.Tokens >= poolConfig.MinDelegationDPoS {
			pool = types.PoolDPoS
		} else {
			pool = types.PoolPoS
		}

		pc := types.PoolClassification{
			ValidatorAddr: val.Address,
			Pool:          pool,
			AssignedAt:    height,
		}
		classifications = append(classifications, pc)

		// Store in KV
		k.SetPoolClassification(ctx, pc)
	}

	// Update stats
	stats := k.GetStats(ctx)
	stats.PoolClassifications++
	k.SetStats(ctx, stats)

	return classifications
}

// SetPoolClassification stores a validator's pool classification.
func (k Keeper) SetPoolClassification(ctx sdk.Context, pc types.PoolClassification) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(pc)
	store.Set(types.PoolClassificationKey(pc.ValidatorAddr), bz)
}

// GetPoolClassification retrieves a validator's pool classification.
func (k Keeper) GetPoolClassification(ctx sdk.Context, validatorAddr string) (types.PoolClassification, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PoolClassificationKey(validatorAddr))
	if bz == nil {
		return types.PoolClassification{}, false
	}
	var pc types.PoolClassification
	if err := json.Unmarshal(bz, &pc); err != nil {
		return types.PoolClassification{}, false
	}
	return pc, true
}

// percentile returns the value at the given percentile (0-100) from a sorted slice.
func percentile(sorted []float64, pct uint64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if pct >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := int(float64(len(sorted)-1) * float64(pct) / 100.0)
	return sorted[idx]
}

// median returns the median value from a sorted slice of uint64.
func median(sorted []uint64) uint64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}
