package types

import "fmt"

// GenesisState defines the babylon module genesis state.
type GenesisState struct {
	Config    BTCRestakingConfig   `json:"config"`
	Positions []BTCStakingPosition `json:"positions"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Config:    DefaultBTCRestakingConfig(),
		Positions: []BTCStakingPosition{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Config.MinStakeAmount <= 0 {
		return fmt.Errorf("min stake amount must be positive")
	}
	if gs.Config.UnbondingPeriod <= 0 {
		return fmt.Errorf("unbonding period must be positive")
	}
	if gs.Config.CheckpointInterval <= 0 {
		return fmt.Errorf("checkpoint interval must be positive")
	}
	return nil
}
