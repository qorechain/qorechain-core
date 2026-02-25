package types

import "fmt"

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
