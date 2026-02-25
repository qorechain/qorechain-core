package types

import "fmt"

// GenesisState defines the inflation module's genesis state.
type GenesisState struct {
	Params    Params    `json:"params"`
	EpochInfo EpochInfo `json:"epoch_info"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		EpochInfo: DefaultEpochInfo(),
	}
}

// Validate performs basic validation of the genesis state.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	if gs.EpochInfo.TotalMinted.IsNegative() {
		return fmt.Errorf("total_minted must be non-negative")
	}
	return nil
}
