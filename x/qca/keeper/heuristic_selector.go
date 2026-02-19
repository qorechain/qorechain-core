package keeper

import (
	"encoding/binary"
	"math"

	"crypto/sha256"

	"github.com/qorechain/qorechain-core/x/qca/types"
)

// HeuristicSelector implements the ValidatorSelector interface using
// reputation-weighted random selection (deterministic sortition).
type HeuristicSelector struct{}

var _ types.ValidatorSelector = (*HeuristicSelector)(nil)

func NewHeuristicSelector() *HeuristicSelector {
	return &HeuristicSelector{}
}

// SelectProposer selects a proposer using weighted random selection.
// Weights = reputation_score * stake. Deterministic using blockHash+height as seed.
func (h *HeuristicSelector) SelectProposer(
	validators []types.ValidatorInfo,
	scores map[string]float64,
	blockHash []byte,
	height int64,
) string {
	if len(validators) == 0 {
		return ""
	}

	// Calculate weights: reputation * stake
	type weightedVal struct {
		address string
		weight  float64
	}
	var weighted []weightedVal
	var totalWeight float64

	for _, val := range validators {
		if !val.Active {
			continue
		}
		score := scores[val.Address]
		if score <= 0 {
			score = 0.1 // Minimum score
		}
		w := score * math.Max(float64(val.Tokens), 1.0)
		weighted = append(weighted, weightedVal{address: val.Address, weight: w})
		totalWeight += w
	}

	if len(weighted) == 0 || totalWeight == 0 {
		return ""
	}

	// Deterministic random using block hash + height
	seed := sha256.New()
	seed.Write(blockHash)
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, uint64(height))
	seed.Write(heightBytes)
	hash := seed.Sum(nil)

	// Convert first 8 bytes to a float64 in [0, 1)
	randVal := float64(binary.LittleEndian.Uint64(hash[:8])) / float64(math.MaxUint64)
	target := randVal * totalWeight

	// Select validator proportional to weight
	var cumulative float64
	for _, wv := range weighted {
		cumulative += wv.weight
		if cumulative >= target {
			return wv.address
		}
	}

	// Fallback to last
	return weighted[len(weighted)-1].address
}
