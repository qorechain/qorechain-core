package types

// GenesisState defines the reputation module's genesis state.
type GenesisState struct {
	Params     ReputationParams      `json:"params"`
	Validators []ValidatorReputation `json:"validators"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:     DefaultReputationParams(),
		Validators: []ValidatorReputation{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	p := gs.Params
	sum := p.Alpha + p.Beta + p.Gamma + p.Delta
	if sum < 0.99 || sum > 1.01 { // Allow small floating point tolerance
		return ErrInvalidParams.Wrapf("weights must sum to 1.0, got %.4f", sum)
	}
	if p.Lambda <= 0 {
		return ErrInvalidParams.Wrap("lambda must be positive")
	}
	return nil
}
