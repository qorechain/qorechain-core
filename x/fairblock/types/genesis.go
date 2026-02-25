package types

import "fmt"

// GenesisState defines the fairblock module genesis state.
type GenesisState struct {
	Config FairBlockConfig `json:"config"`
}

// DefaultGenesisState returns the default genesis state (disabled).
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Config: DefaultFairBlockConfig(),
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Config.TIBEThreshold <= 0 {
		return fmt.Errorf("tIBE threshold must be positive")
	}
	if gs.Config.DecryptionDelay < 0 {
		return fmt.Errorf("decryption delay must be non-negative")
	}
	return nil
}
