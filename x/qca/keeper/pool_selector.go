//go:build proprietary

package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"math"

	cosmosmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/qca/types"
)

// PoolWeightedSelector implements ValidatorSelector with triple-pool weighted selection.
// When used via the ValidatorSelector interface (which lacks sdk.Context), it falls
// back to the inner HeuristicSelector. The full pool-aware selection is performed
// by Keeper.GetPoolWeightedProposer which has access to the KV store.
type PoolWeightedSelector struct {
	inner *HeuristicSelector
}

var _ types.ValidatorSelector = (*PoolWeightedSelector)(nil)

// NewPoolWeightedSelector creates a new PoolWeightedSelector wrapping HeuristicSelector.
func NewPoolWeightedSelector() *PoolWeightedSelector {
	return &PoolWeightedSelector{
		inner: NewHeuristicSelector(),
	}
}

// SelectProposer falls back to the inner HeuristicSelector since the
// ValidatorSelector interface does not provide sdk.Context needed for
// KV store access. Use Keeper.GetPoolWeightedProposer for full pool-aware selection.
func (s *PoolWeightedSelector) SelectProposer(
	validators []types.ValidatorInfo,
	scores map[string]float64,
	blockHash []byte,
	height int64,
) string {
	return s.inner.SelectProposer(validators, scores, blockHash, height)
}

// GetPoolWeightedProposer performs pool-weighted proposer selection.
// This replaces GetReputationWeightedProposer when pool classification is enabled.
//
// Algorithm:
//  1. Group validators by their stored pool classification
//  2. Select a pool via deterministic weighted random (seed = SHA256(blockHash || height || "pool"))
//  3. Within the selected pool, use HeuristicSelector (reputation x stake weighted)
func (k Keeper) GetPoolWeightedProposer(ctx sdk.Context, validators []types.ValidatorInfo) string {
	config := k.GetConfig(ctx)
	poolConfig := config.PoolConfig

	// Parse pool weights
	weightRPoS, err := cosmosmath.LegacyNewDecFromStr(poolConfig.WeightRPoS)
	if err != nil {
		return k.GetReputationWeightedProposer(ctx, validators)
	}
	weightDPoS, err := cosmosmath.LegacyNewDecFromStr(poolConfig.WeightDPoS)
	if err != nil {
		return k.GetReputationWeightedProposer(ctx, validators)
	}

	// Group validators by pool
	pools := map[types.PoolType][]types.ValidatorInfo{
		types.PoolRPoS: {},
		types.PoolDPoS: {},
		types.PoolPoS:  {},
	}
	for _, val := range validators {
		if !val.Active {
			continue
		}
		pc, found := k.GetPoolClassification(ctx, val.Address)
		if !found {
			// Unclassified validators default to PoS
			pools[types.PoolPoS] = append(pools[types.PoolPoS], val)
			continue
		}
		pools[pc.Pool] = append(pools[pc.Pool], val)
	}

	// Deterministic pool selection using SHA256(blockHash || height || "pool")
	blockHash := ctx.BlockHeader().LastBlockId.Hash
	seed := sha256.New()
	seed.Write(blockHash)
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, uint64(ctx.BlockHeight()))
	seed.Write(heightBytes)
	seed.Write([]byte("pool"))
	hash := seed.Sum(nil)

	// Convert first 8 bytes to a float64 in [0, 1)
	randVal := float64(binary.LittleEndian.Uint64(hash[:8])) / float64(math.MaxUint64)

	// Select pool based on weights
	rpos := weightRPoS.MustFloat64()
	dpos := weightDPoS.MustFloat64()

	var selectedPool types.PoolType
	if randVal < rpos {
		selectedPool = types.PoolRPoS
	} else if randVal < rpos+dpos {
		selectedPool = types.PoolDPoS
	} else {
		selectedPool = types.PoolPoS
	}

	// If selected pool is empty, fall back to PoS then all validators
	poolVals := pools[selectedPool]
	if len(poolVals) == 0 {
		poolVals = pools[types.PoolPoS]
	}
	if len(poolVals) == 0 {
		return k.GetReputationWeightedProposer(ctx, validators)
	}

	// Within-pool selection using HeuristicSelector (reputation x stake weighted)
	scores := make(map[string]float64, len(poolVals))
	for _, val := range poolVals {
		scores[val.Address] = k.reputationKeeper.CalculateReputation(ctx, val.Address)
	}

	selector := NewHeuristicSelector()
	selected := selector.SelectProposer(poolVals, scores, blockHash, ctx.BlockHeight())

	// Update stats
	stats := k.GetStats(ctx)
	stats.ProposerSelections++
	stats.ReputationWeighted++
	k.SetStats(ctx, stats)

	return selected
}
