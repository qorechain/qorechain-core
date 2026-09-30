package types

import (
	"fmt"
	"math/bits"
)

// MLPConfig describes the architecture of the multi-layer perceptron used by the
// RL policy network. Input size matches ObservationDimensions and output size
// matches ActionDimensions.
type MLPConfig struct {
	InputSize   int   `json:"input_size"`
	HiddenSizes []int `json:"hidden_sizes"`
	OutputSize  int   `json:"output_size"`
}

// DefaultMLPConfig returns the default MLP architecture:
// 25 input -> [256, 256] hidden -> 5 output.
func DefaultMLPConfig() MLPConfig {
	return MLPConfig{
		InputSize:   ObservationDimensions,
		HiddenSizes: []int{256, 256},
		OutputSize:  ActionDimensions,
	}
}

// TotalParams computes the total number of trainable parameters (weights + biases)
// in the MLP. For the default config: 25*256+256 + 256*256+256 + 256*5+5 = 73,733.
func (c MLPConfig) TotalParams() int {
	if len(c.HiddenSizes) == 0 {
		// Direct input -> output connection.
		return c.InputSize*c.OutputSize + c.OutputSize
	}

	total := 0
	prev := c.InputSize
	for _, hidden := range c.HiddenSizes {
		total += prev*hidden + hidden // weights + biases
		prev = hidden
	}
	total += prev*c.OutputSize + c.OutputSize // final layer
	return total
}

// PolicyWeights holds the flattened weight vector for the RL policy network.
// Weights are stored as int64 fixed-point values with FixedPointScale precision.
type PolicyWeights struct {
	Epoch     uint64    `json:"epoch"`
	Config    MLPConfig `json:"config"`
	Weights   []int64   `json:"weights"`
	UpdatedAt int64     `json:"updated_at"`
}

// Validate checks that the weight vector length matches the MLP configuration.
func (pw PolicyWeights) Validate() error {
	expected := pw.Config.TotalParams()
	if len(pw.Weights) != expected {
		return fmt.Errorf("policy weights length %d does not match config total params %d", len(pw.Weights), expected)
	}
	return nil
}

// Architecture bounds for a policy stored by governance (QSR-2026-0012). The
// network the chain runs is 25 -> [256, 256] -> 5, 73,733 parameters; these
// bounds leave generous room for a retrained policy and nothing for one that
// would make EndBlock inference expensive or wrap the parameter count.
const (
	// MaxPolicyHiddenLayers is the largest number of hidden layers accepted.
	MaxPolicyHiddenLayers = 4
	// MaxPolicyLayerWidth is the widest hidden layer accepted.
	MaxPolicyLayerWidth = 1024
	// MaxPolicyParams is the largest total parameter count accepted.
	MaxPolicyParams = 2_000_000
	// MaxPolicyWeightAbs bounds each fixed-point weight to +/-1000.0, far
	// beyond any trained value and small enough that the forward pass cannot
	// overflow its int64 accumulators on bounded observations.
	MaxPolicyWeightAbs = 1000 * FixedPointScale
)

// TotalParamsChecked is TotalParams computed without wrapping: every product
// and sum is checked, and a negative size is refused. TotalParams itself is
// left as it was, because ValidateBasic calls it and ValidateBasic cannot tell
// which height it runs at, so changing its result would change how a
// historical transaction is judged on replay.
func (c MLPConfig) TotalParamsChecked() (uint64, error) {
	layer := func(total, in, out uint64) (uint64, error) {
		hi, w := bits.Mul64(in, out)
		if hi != 0 {
			return 0, fmt.Errorf("parameter count overflows")
		}
		w, carry := bits.Add64(w, out, 0)
		if carry != 0 {
			return 0, fmt.Errorf("parameter count overflows")
		}
		sum, carry := bits.Add64(total, w, 0)
		if carry != 0 {
			return 0, fmt.Errorf("parameter count overflows")
		}
		return sum, nil
	}
	if c.InputSize < 0 || c.OutputSize < 0 {
		return 0, fmt.Errorf("layer sizes must not be negative")
	}
	var (
		total uint64
		err   error
		prev  = uint64(c.InputSize)
	)
	for _, h := range c.HiddenSizes {
		if h < 0 {
			return 0, fmt.Errorf("layer sizes must not be negative")
		}
		if total, err = layer(total, prev, uint64(h)); err != nil {
			return 0, err
		}
		prev = uint64(h)
	}
	return layer(total, prev, uint64(c.OutputSize))
}

// ValidateArchitecture checks that the network is the shape the agent feeds
// and reads (ObservationDimensions in, ActionDimensions out) and within the
// bounds above.
func (c MLPConfig) ValidateArchitecture() error {
	if c.InputSize != ObservationDimensions {
		return fmt.Errorf("input size %d, expected %d (the observation vector)", c.InputSize, ObservationDimensions)
	}
	if c.OutputSize != ActionDimensions {
		return fmt.Errorf("output size %d, expected %d (the action vector)", c.OutputSize, ActionDimensions)
	}
	if n := len(c.HiddenSizes); n < 1 || n > MaxPolicyHiddenLayers {
		return fmt.Errorf("%d hidden layers, expected 1 to %d", n, MaxPolicyHiddenLayers)
	}
	for i, h := range c.HiddenSizes {
		if h < 1 || h > MaxPolicyLayerWidth {
			return fmt.Errorf("hidden layer %d has width %d, expected 1 to %d", i, h, MaxPolicyLayerWidth)
		}
	}
	total, err := c.TotalParamsChecked()
	if err != nil {
		return err
	}
	if total > MaxPolicyParams {
		return fmt.Errorf("%d parameters, at most %d accepted", total, MaxPolicyParams)
	}
	return nil
}

// ValidateStrict is the full check a policy must pass before the chain stores
// it or runs it: the architecture, the exact weight count, and each weight's
// range.
func (pw PolicyWeights) ValidateStrict() error {
	if err := pw.Config.ValidateArchitecture(); err != nil {
		return err
	}
	total, _ := pw.Config.TotalParamsChecked() // checked by ValidateArchitecture
	if uint64(len(pw.Weights)) != total {
		return fmt.Errorf("policy weights length %d does not match config total params %d", len(pw.Weights), total)
	}
	for i, w := range pw.Weights {
		if w > MaxPolicyWeightAbs || w < -MaxPolicyWeightAbs {
			return fmt.Errorf("weight %d is %d, outside +/-%d", i, w, int64(MaxPolicyWeightAbs))
		}
	}
	return nil
}
