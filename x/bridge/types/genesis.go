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
		return fmt.Errorf("attestation threshold must be >= 1")
	}
	if gs.Config.AttestationThreshold > gs.Config.MinValidators {
		return fmt.Errorf("attestation threshold (%d) cannot exceed min validators (%d)",
			gs.Config.AttestationThreshold, gs.Config.MinValidators)
	}
	if _, err := ParseStoredAmount(gs.Config.LargeTransferThreshold); err != nil {
		return fmt.Errorf("large_transfer_threshold: %w", err)
	}
	// The same rules the admin messages apply at run time (QSR-2026-0047), so a
	// genesis file cannot install what a transaction would be refused for.
	chains := make(map[string]bool, len(gs.ChainConfigs))
	for _, cc := range gs.ChainConfigs {
		if cc.ChainID == "" {
			return fmt.Errorf("chain config has empty chain_id")
		}
		if err := cc.Validate(); err != nil {
			return err
		}
		if chains[cc.ChainID] {
			return fmt.Errorf("chain config %s appears twice", cc.ChainID)
		}
		chains[cc.ChainID] = true
	}
	lockedSeen := make(map[string]bool, len(gs.Locked))
	for _, la := range gs.Locked {
		if err := la.Validate(); err != nil {
			return err
		}
		key := la.Chain + "/" + la.Asset
		if lockedSeen[key] {
			return fmt.Errorf("locked amount %s appears twice", key)
		}
		lockedSeen[key] = true
	}
	breakers := make(map[string]bool, len(gs.Breakers))
	for _, cb := range gs.Breakers {
		if err := cb.Validate(); err != nil {
			return err
		}
		if breakers[cb.Chain] {
			return fmt.Errorf("circuit breaker %s appears twice", cb.Chain)
		}
		breakers[cb.Chain] = true
	}
	return nil
}
