package types

import "fmt"

// GenesisState defines the bridge module's genesis state.
type GenesisState struct {
	Config       BridgeConfig          `json:"config"`
	ChainConfigs []ChainConfig         `json:"chain_configs"`
	Validators   []BridgeValidator     `json:"validators"`
	Operations   []BridgeOperation     `json:"operations"`
	Locked       []LockedAmount        `json:"locked"`
	Breakers     []CircuitBreakerState `json:"circuit_breakers"`
}

// DefaultGenesisState returns the default genesis state for the bridge module.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Config:       DefaultBridgeConfig(),
		ChainConfigs: DefaultChainConfigs(),
		Validators:   []BridgeValidator{},
		Operations:   []BridgeOperation{},
		Locked:       []LockedAmount{},
		Breakers:     []CircuitBreakerState{},
	}
}

// Validate performs basic validation of the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Config.MinValidators < 1 {
		return fmt.Errorf("min_validators must be >= 1")
	}
	if gs.Config.AttestationThreshold < 1 {
		return fmt.Errorf("attestation_threshold must be >= 1")
	}
	if gs.Config.AttestationThreshold > gs.Config.MinValidators*2 {
		return fmt.Errorf("attestation_threshold cannot exceed 2x min_validators")
	}
	for _, cc := range gs.ChainConfigs {
		if cc.ChainID == "" {
			return fmt.Errorf("chain config has empty chain_id")
		}
		if cc.Name == "" {
			return fmt.Errorf("chain config %s has empty name", cc.ChainID)
		}
	}
	return nil
}
