package types

// GenesisState defines the qca module's genesis state.
type GenesisState struct {
	Config QCAConfig `json:"config"`
	Stats  QCAStats  `json:"stats"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Config: DefaultQCAConfig(),
		Stats:  QCAStats{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	return nil
}
