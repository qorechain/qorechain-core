package types

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

// Validate validates the genesis state with the same rules as the config
// itself: every accepted token needs a parseable, positive conversion rate,
// not merely a non-empty one.
func (gs GenesisState) Validate() error {
	return gs.Config.Validate()
}
