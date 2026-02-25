package types

import "fmt"

// GenesisState defines the gasabstraction module genesis state.
type GenesisState struct {
	Config GasAbstractionConfig `json:"config"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Config: DefaultGasAbstractionConfig(),
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Config.NativeDenom == "" {
		return fmt.Errorf("native denom cannot be empty")
	}
	for _, t := range gs.Config.AcceptedTokens {
		if t.Denom == "" {
			return fmt.Errorf("accepted token denom cannot be empty")
		}
		if t.ConversionRate == "" {
			return fmt.Errorf("conversion rate cannot be empty for denom %s", t.Denom)
		}
	}
	return nil
}
