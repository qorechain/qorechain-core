package types

import "fmt"

// GenesisState defines the burn module's genesis state.
type GenesisState struct {
	Params  Params       `json:"params"`
	Stats   BurnStats    `json:"stats"`
	Records []BurnRecord `json:"records"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Stats:   DefaultBurnStats(),
		Records: []BurnRecord{},
	}
}

// Validate performs basic validation of the genesis state.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	if gs.Stats.TotalBurned.IsNegative() {
		return fmt.Errorf("total_burned must be non-negative")
	}
	for i, r := range gs.Records {
		if !IsValidBurnSource(r.Source) {
			return fmt.Errorf("record[%d]: invalid source %q", i, r.Source)
		}
		if r.Amount.IsNegative() {
			return fmt.Errorf("record[%d]: amount must be non-negative", i)
		}
	}
	return nil
}
