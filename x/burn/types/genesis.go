package types

import "fmt"

// GenesisState defines the burn module's genesis state.
type GenesisState struct {
	Params         Params         `json:"params"`
	Stats          BurnStats      `json:"stats"`
	Records        []BurnRecord   `json:"records"`
	MilestoneState MilestoneState `json:"milestone_state"`
	// StrandedRewards is the governance-releasable stranded staker share
	// recorded by the v3.2.0 release handler (QSR-2026-0473). Absent on a
	// chain that never took the snapshot; files without the field import as
	// before.
	StrandedRewards *StrandedRewards `json:"stranded_rewards,omitempty"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		Stats:          DefaultBurnStats(),
		Records:        []BurnRecord{},
		MilestoneState: DefaultMilestoneState(),
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
	if gs.StrandedRewards != nil {
		if err := gs.StrandedRewards.Validate(); err != nil {
			return fmt.Errorf("invalid stranded_rewards: %w", err)
		}
		if gs.StrandedRewards.SnapshotHeight < 0 {
			return fmt.Errorf("invalid stranded_rewards: snapshot_height must be non-negative")
		}
	}
	return nil
}
