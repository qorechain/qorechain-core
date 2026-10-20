package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// GenesisState defines the lightnode module's genesis state.
type GenesisState struct {
	Params     Params          `json:"params"`
	LightNodes []LightNodeInfo `json:"light_nodes"`
	Stats      LightNodeStats  `json:"stats"`
	// RewardPool is present in the genesis of a chain born on this binary and
	// absent from the genuine genesis of a chain that upgrades into it. Absent
	// means InitGenesis writes no pool record at height 0, exactly like the
	// binary the network's history was made with; the coordinated upgrade
	// handler establishes the record at the plan height instead.
	RewardPool *RewardPool `json:"reward_pool,omitempty"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		LightNodes: []LightNodeInfo{},
		Stats:      DefaultLightNodeStats(),
		RewardPool: func() *RewardPool { p := DefaultRewardPool(); return &p }(),
	}
}

// Validate performs basic validation of the genesis state.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	seen := make(map[string]bool)
	for i, node := range gs.LightNodes {
		if node.Address == "" {
			return fmt.Errorf("light_nodes[%d]: address cannot be empty", i)
		}
		if seen[node.Address] {
			return fmt.Errorf("light_nodes[%d]: duplicate address %s", i, node.Address)
		}
		seen[node.Address] = true
		if !ValidNodeType(node.NodeType) {
			return fmt.Errorf("light_nodes[%d]: invalid node type %q", i, node.NodeType)
		}
		if node.Status != NodeStatusActive && node.Status != NodeStatusInactive {
			return fmt.Errorf("light_nodes[%d]: invalid status %q", i, node.Status)
		}
	}
	if gs.Stats.TotalRewards.IsNegative() {
		return fmt.Errorf("stats total_rewards must be non-negative")
	}
	// Every node's accumulated_rewards is a non-negative integer (empty reads
	// as zero, as the keeper reads it), and their sum is what the pool owes.
	credited := math.ZeroInt()
	for i, node := range gs.LightNodes {
		if node.AccumulatedRewards == "" {
			continue
		}
		amt, ok := math.NewIntFromString(node.AccumulatedRewards)
		if !ok {
			return fmt.Errorf("light_nodes[%d]: accumulated_rewards %q is not an integer", i, node.AccumulatedRewards)
		}
		if amt.IsNegative() {
			return fmt.Errorf("light_nodes[%d]: accumulated_rewards must be non-negative, got %s", i, amt)
		}
		credited = credited.Add(amt)
	}
	if gs.RewardPool != nil {
		if err := gs.RewardPool.Validate(); err != nil {
			return fmt.Errorf("invalid reward_pool: %w", err)
		}
		// QSR-2026-0496: Outstanding is by definition the sum of the nodes'
		// accumulated rewards. A genesis that breaks this starts a chain whose
		// claims are not all backed. An uninitialized record is recomputed
		// from the nodes by the upgrade path, so only an initialized one is
		// held to it.
		if pool := gs.RewardPool.Normalized(); pool.Initialized && !pool.Outstanding.Equal(credited) {
			return fmt.Errorf("invalid reward_pool: outstanding %s does not equal the sum of the nodes' accumulated_rewards %s",
				pool.Outstanding, credited)
		}
	}
	return nil
}
