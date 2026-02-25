package types

import "fmt"

// AgentStatus tracks the current state of the RL agent.
type AgentStatus struct {
	Mode                 AgentMode `json:"mode"`
	CurrentEpoch         uint64    `json:"current_epoch"`
	TotalSteps           uint64    `json:"total_steps"`
	LastObservationAt    int64     `json:"last_observation_at"`
	LastActionAt         int64     `json:"last_action_at"`
	CircuitBreakerActive bool      `json:"circuit_breaker_active"`
	BlocksSinceRevert    int64     `json:"blocks_since_revert"`
}

// GenesisState defines the rlconsensus module's genesis state.
type GenesisState struct {
	Params        Params         `json:"params"`
	AgentStatus   AgentStatus    `json:"agent_status"`
	PolicyWeights *PolicyWeights `json:"policy_weights,omitempty"`
}

// DefaultGenesis returns a genesis state with default parameters,
// shadow mode agent status, and no policy weights.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		AgentStatus: AgentStatus{
			Mode:                 AgentModeShadow,
			CurrentEpoch:         0,
			TotalSteps:           0,
			LastObservationAt:    0,
			LastActionAt:         0,
			CircuitBreakerActive: false,
			BlocksSinceRevert:    0,
		},
		PolicyWeights: nil,
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	if !ValidAgentMode(gs.AgentStatus.Mode) {
		return fmt.Errorf("invalid agent mode in status: %d", gs.AgentStatus.Mode)
	}
	if gs.PolicyWeights != nil {
		if err := gs.PolicyWeights.Validate(); err != nil {
			return fmt.Errorf("invalid policy weights: %w", err)
		}
	}
	return nil
}
