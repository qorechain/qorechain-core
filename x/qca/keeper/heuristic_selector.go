package keeper

import (
	"encoding/binary"
	"fmt"
	"math"

	"crypto/sha256"

	sdkmath "cosmossdk.io/math"

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
) (string, error) {
	if len(validators) == 0 {
		return "", types.ErrNoValidators
	}

	// Calculate weights: reputation * stake (using LegacyDec for determinism)
	type weightedVal struct {
		address string
		weight  sdkmath.LegacyDec
	}
	var weighted []weightedVal
	totalWeight := sdkmath.LegacyZeroDec()

	for _, val := range validators {
		if !val.Active {
			continue
		}
		score := scores[val.Address]
		if score <= 0 {
			continue // exclude zero/negative reputation validators from selection
		}
		scoreDec := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.18f", score))
		tokensDec := sdkmath.LegacyNewDecFromInt(sdkmath.NewIntFromUint64(max(val.Tokens, 1)))
		w := scoreDec.Mul(tokensDec)
		weighted = append(weighted, weightedVal{address: val.Address, weight: w})
		totalWeight = totalWeight.Add(w)
	}

	if len(weighted) == 0 || totalWeight.IsZero() {
		return "", types.ErrNoValidators
	}

	// Deterministic random using block hash + height
	seed := sha256.New()
	seed.Write(blockHash)
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, uint64(height))
	seed.Write(heightBytes)
	hash := seed.Sum(nil)

	// Convert first 8 bytes to a LegacyDec in [0, 1) for deterministic selection
	randInt := sdkmath.NewIntFromUint64(binary.LittleEndian.Uint64(hash[:8]))
	maxInt := sdkmath.NewIntFromUint64(math.MaxUint64)
	randDec := sdkmath.LegacyNewDecFromInt(randInt).Quo(sdkmath.LegacyNewDecFromInt(maxInt))
	target := randDec.Mul(totalWeight)

	// Select validator proportional to weight
	cumulative := sdkmath.LegacyZeroDec()
	for _, wv := range weighted {
		cumulative = cumulative.Add(wv.weight)
		if cumulative.GTE(target) {
			return wv.address, nil
		}
	}

	// Fallback to last
	return weighted[len(weighted)-1].address, nil
}
