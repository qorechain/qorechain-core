package types

import sdkmath "cosmossdk.io/math"

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
	alpha := p.ParamAlpha()
	beta := p.ParamBeta()
	gamma := p.ParamGamma()
	delta := p.ParamDelta()
	lambda := p.ParamLambda()

	sum := alpha.Add(beta).Add(gamma).Add(delta)
	lower := sdkmath.LegacyNewDecWithPrec(99, 2)  // 0.99
	upper := sdkmath.LegacyNewDecWithPrec(101, 2) // 1.01
	if sum.LT(lower) || sum.GT(upper) {
		return ErrInvalidParams.Wrapf("weights must sum to 1.0, got %s", sum.String())
	}
	if !lambda.IsPositive() {
		return ErrInvalidParams.Wrap("lambda must be positive")
	}
	return nil
}
