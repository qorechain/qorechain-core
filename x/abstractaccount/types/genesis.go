package types

import "fmt"

// GenesisState defines the abstractaccount module genesis state.
type GenesisState struct {
	Config   AbstractAccountConfig `json:"config"`
	Accounts []AbstractAccount     `json:"accounts"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Config:   DefaultAbstractAccountConfig(),
		Accounts: []AbstractAccount{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Config.MaxSessionKeys <= 0 {
		return fmt.Errorf("max session keys must be positive")
	}
	if gs.Config.MaxSpendingRules <= 0 {
		return fmt.Errorf("max spending rules must be positive")
	}
	if gs.Config.DefaultSessionTTL <= 0 {
		return fmt.Errorf("default session TTL must be positive")
	}
	return nil
}
