package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// AgentMode represents the operating mode of the RL agent.
type AgentMode uint8

const (
	// AgentModeShadow logs recommendations without applying them.
	AgentModeShadow AgentMode = 0
	// AgentModeConservative applies changes within tight bounds.
	AgentModeConservative AgentMode = 1
	// AgentModeAutonomous applies changes within wider bounds.
	AgentModeAutonomous AgentMode = 2
	// AgentModePaused disables observation and action entirely.
	AgentModePaused AgentMode = 3
)

// String returns the human-readable name of an AgentMode.
func (m AgentMode) String() string {
	switch m {
	case AgentModeShadow:
		return "shadow"
	case AgentModeConservative:
		return "conservative"
	case AgentModeAutonomous:
		return "autonomous"
	case AgentModePaused:
		return "paused"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

// ValidAgentMode returns true if the mode is a recognized value.
func ValidAgentMode(m AgentMode) bool {
	return m <= AgentModePaused
}

// Default parameter values.
const (
	DefaultEnabled              = true
	DefaultObservationInterval  = uint64(10)
	DefaultBlockTimeMs          = int64(5000)
	DefaultValidatorSetSize     = uint64(100)
	DefaultCircuitBreakerWindow = uint64(50)
)

// Default string representations for LegacyDec fields.
const (
	DefaultMaxChangeConservative     = "0.10"
	DefaultMaxChangeAutonomous       = "0.25"
	DefaultCircuitBreakerThreshold   = "0.50"
	DefaultBaseGasPrice              = "100"
	DefaultRewardWeightThroughput    = "0.30"
	DefaultRewardWeightFinality      = "0.25"
	DefaultRewardWeightDecentralization = "0.20"
	DefaultRewardWeightMEV           = "0.15"
	DefaultRewardWeightFailedTxs     = "0.10"
)

// RewardWeights defines the relative importance of each reward component.
// All weights must sum to exactly 1.0.
type RewardWeights struct {
	Throughput       string `json:"throughput"`
	Finality         string `json:"finality"`
	Decentralization string `json:"decentralization"`
	MEV              string `json:"mev"`
	FailedTxs        string `json:"failed_txs"`
}

// DefaultRewardWeights returns the default reward weight configuration.
func DefaultRewardWeights() RewardWeights {
	return RewardWeights{
		Throughput:       DefaultRewardWeightThroughput,
		Finality:         DefaultRewardWeightFinality,
		Decentralization: DefaultRewardWeightDecentralization,
		MEV:              DefaultRewardWeightMEV,
		FailedTxs:        DefaultRewardWeightFailedTxs,
	}
}

// Validate checks that all reward weights are valid decimals and sum to 1.0.
func (rw RewardWeights) Validate() error {
	t, err := math.LegacyNewDecFromStr(rw.Throughput)
	if err != nil {
		return fmt.Errorf("invalid throughput weight: %w", err)
	}
	f, err := math.LegacyNewDecFromStr(rw.Finality)
	if err != nil {
		return fmt.Errorf("invalid finality weight: %w", err)
	}
	d, err := math.LegacyNewDecFromStr(rw.Decentralization)
	if err != nil {
		return fmt.Errorf("invalid decentralization weight: %w", err)
	}
	m, err := math.LegacyNewDecFromStr(rw.MEV)
	if err != nil {
		return fmt.Errorf("invalid MEV weight: %w", err)
	}
	ft, err := math.LegacyNewDecFromStr(rw.FailedTxs)
	if err != nil {
		return fmt.Errorf("invalid failed_txs weight: %w", err)
	}

	sum := t.Add(f).Add(d).Add(m).Add(ft)
	one := math.LegacyOneDec()
	if !sum.Equal(one) {
		return fmt.Errorf("reward weights must sum to 1.0, got %s", sum.String())
	}
	return nil
}

// Params defines the configurable parameters for the RL consensus module.
type Params struct {
	Enabled                 bool         `json:"enabled"`
	ObservationInterval     uint64       `json:"observation_interval"`
	AgentMode               AgentMode    `json:"agent_mode"`
	MaxChangeConservative   string       `json:"max_change_conservative"`
	MaxChangeAutonomous     string       `json:"max_change_autonomous"`
	CircuitBreakerWindow    uint64       `json:"circuit_breaker_window"`
	CircuitBreakerThreshold string       `json:"circuit_breaker_threshold"`
	RewardWeights           RewardWeights `json:"reward_weights"`
	DefaultBlockTimeMs      int64        `json:"default_block_time_ms"`
	DefaultBaseGasPrice     string       `json:"default_base_gas_price"`
	DefaultValidatorSetSize uint64       `json:"default_validator_set_size"`
}

// DefaultParams returns a default set of RL consensus parameters.
func DefaultParams() Params {
	return Params{
		Enabled:                 DefaultEnabled,
		ObservationInterval:     DefaultObservationInterval,
		AgentMode:               AgentModeShadow,
		MaxChangeConservative:   DefaultMaxChangeConservative,
		MaxChangeAutonomous:     DefaultMaxChangeAutonomous,
		CircuitBreakerWindow:    DefaultCircuitBreakerWindow,
		CircuitBreakerThreshold: DefaultCircuitBreakerThreshold,
		RewardWeights:           DefaultRewardWeights(),
		DefaultBlockTimeMs:      DefaultBlockTimeMs,
		DefaultBaseGasPrice:     DefaultBaseGasPrice,
		DefaultValidatorSetSize: DefaultValidatorSetSize,
	}
}

// Validate performs basic validation of RL consensus parameters.
func (p Params) Validate() error {
	if p.ObservationInterval < 1 {
		return fmt.Errorf("observation interval must be >= 1, got %d", p.ObservationInterval)
	}
	if !ValidAgentMode(p.AgentMode) {
		return fmt.Errorf("invalid agent mode: %d", p.AgentMode)
	}
	if _, err := math.LegacyNewDecFromStr(p.MaxChangeConservative); err != nil {
		return fmt.Errorf("invalid max_change_conservative: %w", err)
	}
	if _, err := math.LegacyNewDecFromStr(p.MaxChangeAutonomous); err != nil {
		return fmt.Errorf("invalid max_change_autonomous: %w", err)
	}
	if p.CircuitBreakerWindow < 10 {
		return fmt.Errorf("circuit breaker window must be >= 10, got %d", p.CircuitBreakerWindow)
	}
	if _, err := math.LegacyNewDecFromStr(p.CircuitBreakerThreshold); err != nil {
		return fmt.Errorf("invalid circuit_breaker_threshold: %w", err)
	}
	if _, err := math.LegacyNewDecFromStr(p.DefaultBaseGasPrice); err != nil {
		return fmt.Errorf("invalid default_base_gas_price: %w", err)
	}
	if err := p.RewardWeights.Validate(); err != nil {
		return err
	}
	return nil
}

// MaxChangeForMode returns the maximum parameter change allowed for the
// current agent mode as a LegacyDec. Shadow and Paused modes return zero
// (no changes applied).
func (p Params) MaxChangeForMode() math.LegacyDec {
	switch p.AgentMode {
	case AgentModeConservative:
		d, err := math.LegacyNewDecFromStr(p.MaxChangeConservative)
		if err != nil {
			return math.LegacyZeroDec()
		}
		return d
	case AgentModeAutonomous:
		d, err := math.LegacyNewDecFromStr(p.MaxChangeAutonomous)
		if err != nil {
			return math.LegacyZeroDec()
		}
		return d
	default:
		// Shadow and Paused: no changes applied.
		return math.LegacyZeroDec()
	}
}
