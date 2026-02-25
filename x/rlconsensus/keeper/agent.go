//go:build proprietary

package keeper

import (
	"fmt"

	"cosmossdk.io/math"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// PPOAgent wraps the MLP policy network and converts between
// Observation/Action types and fixed-point vectors.
type PPOAgent struct {
	mlp    *MLP
	config types.MLPConfig
}

// NewPPOAgent creates a PPO agent from stored policy weights.
func NewPPOAgent(pw *types.PolicyWeights) *PPOAgent {
	mlp := NewMLP(pw.Config, pw.Weights)
	return &PPOAgent{
		mlp:    mlp,
		config: pw.Config,
	}
}

// Infer runs the MLP forward pass on an observation and returns an action vector.
func (a *PPOAgent) Infer(obs *types.Observation) (*types.Action, error) {
	// Convert observation to fixed-point
	fixedInput, err := obs.ToFixedPoint()
	if err != nil {
		return nil, fmt.Errorf("failed to convert observation to fixed-point: %w", err)
	}

	// Run forward pass
	inputSlice := fixedInput[:]
	output, err := a.mlp.Forward(inputSlice)
	if err != nil {
		return nil, fmt.Errorf("MLP forward pass failed: %w", err)
	}

	if len(output) != types.ActionDimensions {
		return nil, fmt.Errorf("MLP output size %d does not match ActionDimensions %d", len(output), types.ActionDimensions)
	}

	// Convert fixed-point output back to LegacyDec strings
	action := &types.Action{
		Height: obs.Height,
	}

	scale := math.LegacyNewDec(FixedPointScale)
	for i := 0; i < types.ActionDimensions; i++ {
		// output[i] is in fixed-point (value * SCALE)
		// Convert back: value = output[i] / SCALE
		val := math.LegacyNewDec(output[i]).Quo(scale)
		action.Values[i] = val.String()
	}

	return action, nil
}
