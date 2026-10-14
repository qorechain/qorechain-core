package types

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
)

func genesisWithNodes(rewards ...string) *GenesisState {
	gs := DefaultGenesisState()
	for i, r := range rewards {
		gs.LightNodes = append(gs.LightNodes, LightNodeInfo{
			Address:            string(rune('a'+i)) + "-node",
			NodeType:           NodeTypeSX,
			Status:             NodeStatusActive,
			AccumulatedRewards: r,
		})
	}
	return gs
}

// QSR-2026-0496: an initialized reward pool's outstanding must equal the sum
// of the nodes' accumulated rewards, and every node's figure must parse.
func TestGenesisRewardPoolOutstandingMatchesNodes(t *testing.T) {
	gs := genesisWithNodes("100", "", "250")
	gs.RewardPool.Outstanding = math.NewInt(350)
	if err := gs.Validate(); err != nil {
		t.Fatalf("consistent genesis refused: %v", err)
	}

	gs.RewardPool.Outstanding = math.NewInt(0)
	if err := gs.Validate(); err == nil || !strings.Contains(err.Error(), "outstanding") {
		t.Fatalf("mismatched outstanding accepted: %v", err)
	}

	// An uninitialized pool is recomputed from the nodes later, so it is not
	// held to the sum.
	gs.RewardPool.Initialized = false
	if err := gs.Validate(); err != nil {
		t.Fatalf("uninitialized pool refused: %v", err)
	}

	// A genesis without a pool (an upgraded chain's original genesis) only
	// needs parseable figures.
	gs.RewardPool = nil
	if err := gs.Validate(); err != nil {
		t.Fatalf("pool-less genesis refused: %v", err)
	}

	for _, bad := range []string{"-5", "1.5", "lots"} {
		g := genesisWithNodes(bad)
		g.RewardPool.Outstanding = math.ZeroInt()
		if err := g.Validate(); err == nil {
			t.Errorf("accumulated_rewards %q accepted", bad)
		}
	}
}
