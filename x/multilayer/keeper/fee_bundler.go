//go:build proprietary

package keeper

import (
	"fmt"
	"strconv"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CalculateCrossLayerFee computes a bundled fee for cross-layer operations (CLFB).
// A single fee paid on the source layer covers execution across all layers in the TX path.
// Fee distribution is proportional to estimated gas consumed on each layer.
func (k Keeper) CalculateCrossLayerFee(ctx sdk.Context, sourceLayers []string, totalGas uint64) (sdk.Coins, error) {
	params := k.GetParams(ctx)

	// Check if CLFB is enabled globally
	if !params.CrossLayerFeeBundling {
		return nil, fmt.Errorf("cross-layer fee bundling (CLFB) is disabled")
	}

	if len(sourceLayers) == 0 {
		return nil, fmt.Errorf("no source layers specified")
	}

	// Calculate total fee multiplier across all layers
	totalMultiplier := 0.0
	for _, layerID := range sourceLayers {
		if layerID == "main" {
			totalMultiplier += 1.0 // Main chain base multiplier
			continue
		}

		layer, err := k.GetLayer(ctx, layerID)
		if err != nil {
			return nil, fmt.Errorf("layer %s not found: %w", layerID, err)
		}

		if !layer.CrossLayerFeeBundlingEnabled {
			return nil, fmt.Errorf("layer %s does not support CLFB", layerID)
		}

		multiplier, err := strconv.ParseFloat(layer.BaseFeeMultiplier, 64)
		if err != nil {
			multiplier = 1.0 // Default to main chain rate
		}
		totalMultiplier += multiplier
	}

	// Average multiplier across all layers
	avgMultiplier := totalMultiplier / float64(len(sourceLayers))

	// Calculate bundled fee in uqor
	// Base fee: 1 uqor per 1000 gas, adjusted by average layer multiplier
	baseFee := float64(totalGas) / 1000.0 * avgMultiplier
	feeAmount := int64(baseFee)
	if feeAmount < 1 {
		feeAmount = 1 // Minimum fee of 1 uqor
	}

	bundledFee := sdk.NewCoins(sdk.NewCoin("uqor", sdkmath.NewInt(feeAmount)))

	k.logger.Info("CLFB fee calculated",
		"layers", sourceLayers,
		"total_gas", totalGas,
		"avg_multiplier", fmt.Sprintf("%.4f", avgMultiplier),
		"bundled_fee", bundledFee.String(),
	)

	return bundledFee, nil
}
