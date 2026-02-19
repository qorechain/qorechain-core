package types

// GenesisState defines the pqc module's genesis state.
type GenesisState struct {
	Params   Params           `json:"params"`
	Accounts []PQCAccountInfo `json:"accounts"`
	Stats    PQCStats         `json:"stats"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Accounts: []PQCAccountInfo{},
		Stats:    PQCStats{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Params.MinSecurityLevel < 1 || gs.Params.MinSecurityLevel > 5 {
		return ErrInvalidKeyLength.Wrap("min_security_level must be between 1 and 5")
	}
	return nil
}
