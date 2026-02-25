//go:build proprietary

package keeper

import (
	"fmt"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// FixedPointScale matches types.FixedPointScale (10^8).
const FixedPointScale int64 = types.FixedPointScale

// MLP is a fixed-point multi-layer perceptron.
// All computations use int64 scaled by FixedPointScale to ensure determinism.
type MLP struct {
	config  types.MLPConfig
	weights []int64 // flattened: [layer0_weights, layer0_biases, layer1_weights, ...]
}

// NewMLP creates an MLP from a configuration and flattened weight vector.
func NewMLP(config types.MLPConfig, weights []int64) *MLP {
	return &MLP{
		config:  config,
		weights: weights,
	}
}

// Forward performs a forward pass: input -> hidden layers (ReLU) -> output (tanh).
// All arithmetic is in fixed-point with FixedPointScale precision.
func (m *MLP) Forward(input []int64) ([]int64, error) {
	if len(input) != m.config.InputSize {
		return nil, fmt.Errorf("MLP input size mismatch: expected %d, got %d", m.config.InputSize, len(input))
	}

	current := make([]int64, len(input))
	copy(current, input)

	offset := 0
	prevSize := m.config.InputSize

	// Process hidden layers with ReLU activation
	for layerIdx, hiddenSize := range m.config.HiddenSizes {
		weightsNeeded := prevSize * hiddenSize
		biasesNeeded := hiddenSize

		if offset+weightsNeeded+biasesNeeded > len(m.weights) {
			return nil, fmt.Errorf("MLP weight buffer underrun at hidden layer %d", layerIdx)
		}

		layerWeights := m.weights[offset : offset+weightsNeeded]
		offset += weightsNeeded
		layerBiases := m.weights[offset : offset+biasesNeeded]
		offset += biasesNeeded

		output := make([]int64, hiddenSize)
		for j := 0; j < hiddenSize; j++ {
			sum := layerBiases[j]
			for i := 0; i < prevSize; i++ {
				// Fixed-point multiply: (a * b) / SCALE
				prod := fixMul(current[i], layerWeights[i*hiddenSize+j])
				sum += prod
			}
			// ReLU activation
			output[j] = relu(sum)
		}

		current = output
		prevSize = hiddenSize
	}

	// Output layer with tanh activation
	outSize := m.config.OutputSize
	weightsNeeded := prevSize * outSize
	biasesNeeded := outSize

	if offset+weightsNeeded+biasesNeeded > len(m.weights) {
		return nil, fmt.Errorf("MLP weight buffer underrun at output layer")
	}

	outWeights := m.weights[offset : offset+weightsNeeded]
	offset += weightsNeeded
	outBiases := m.weights[offset : offset+biasesNeeded]

	output := make([]int64, outSize)
	for j := 0; j < outSize; j++ {
		sum := outBiases[j]
		for i := 0; i < prevSize; i++ {
			prod := fixMul(current[i], outWeights[i*outSize+j])
			sum += prod
		}
		// Tanh activation
		output[j] = tanhApprox(sum)
	}

	return output, nil
}

// fixMul performs fixed-point multiplication: (a * b) / SCALE.
// Uses int64 with overflow check via intermediate split.
func fixMul(a, b int64) int64 {
	// Split to avoid overflow: a*b can overflow int64 for large values.
	// Use the identity: a*b/S = (a/S)*b + (a%S)*b/S
	hi := (a / FixedPointScale) * b
	lo := (a % FixedPointScale) * b / FixedPointScale
	return hi + lo
}

// relu applies the ReLU activation function.
func relu(x int64) int64 {
	if x > 0 {
		return x
	}
	return 0
}

// tanhApprox computes a fixed-point approximation of tanh(x/SCALE)*SCALE.
// Uses a rational Pade approximant for |x| <= 2.5*SCALE:
//
//	tanh(x) ~ x * (1 - x^2 / (3*SCALE^2)) / (1 + x^2 / (3*SCALE^2))
//
// For |x| > 2.5*SCALE, clamps to +/- SCALE.
func tanhApprox(x int64) int64 {
	scale := FixedPointScale
	limit := scale * 5 / 2 // 2.5 * SCALE

	// Clamp for large values
	if x > limit {
		return scale
	}
	if x < -limit {
		return -scale
	}

	// Pade approximant: tanh(x) ~ x * (3*S^2 - x^2) / (3*S^2 + x^2)
	// Working in reduced precision to avoid overflow:
	// Let xr = x (already in fixed-point, so x represents x/SCALE in real)
	// x^2 in fixed-point: fixMul(x, x)
	x2 := fixMul(x, x)

	// 3 * SCALE (represents 3.0 in fixed-point)
	threeScale := 3 * scale

	// numerator = x * (3*SCALE - x2/SCALE)
	// denominator = (3*SCALE + x2/SCALE)
	// Note: x2 is already x^2 / SCALE, so x2/SCALE = x^2 / SCALE^2
	// We need x^2 / (3*SCALE^2) but rewriting:
	// tanh ~ x * (3*S - x2) / (3*S + x2)  where x2 = x*x/S
	num := threeScale - x2
	den := threeScale + x2

	if den == 0 {
		if x > 0 {
			return scale
		}
		return -scale
	}

	// result = x * num / den
	return fixMul(x, num) * scale / den
}
