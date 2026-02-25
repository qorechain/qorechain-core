package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// GenesisState defines the xQORE module's genesis state.
type GenesisState struct {
	Params      Params          `json:"params"`
	Positions   []XQOREPosition `json:"positions"`
	TotalLocked math.Int        `json:"total_locked"`
	TotalXQORE  math.Int        `json:"total_xqore"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Positions:   []XQOREPosition{},
		TotalLocked: math.ZeroInt(),
		TotalXQORE:  math.ZeroInt(),
	}
}

// Validate performs basic validation of the genesis state.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	if gs.TotalLocked.IsNegative() {
		return fmt.Errorf("total_locked must be non-negative")
	}
	if gs.TotalXQORE.IsNegative() {
		return fmt.Errorf("total_xqore must be non-negative")
	}
	for i, pos := range gs.Positions {
		if pos.Owner == "" {
			return fmt.Errorf("position[%d]: owner must not be empty", i)
		}
		if pos.Locked.IsNegative() {
			return fmt.Errorf("position[%d]: locked must be non-negative", i)
		}
		if pos.XBalance.IsNegative() {
			return fmt.Errorf("position[%d]: x_balance must be non-negative", i)
		}
	}
	return nil
}
