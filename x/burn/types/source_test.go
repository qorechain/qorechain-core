package types

import "testing"

// The staking tier penalty is a valid source, and DefaultBurnStats lists it,
// so a fresh chain reports it from the first block.
func TestStakingTierPenaltySource(t *testing.T) {
	if !IsValidBurnSource(BurnSourceStakingTierPenalty) {
		t.Fatal("staking_tier_penalty must be a valid burn source")
	}
	if _, ok := DefaultBurnStats().BurnsBySource[BurnSourceStakingTierPenalty]; !ok {
		t.Fatal("default burn stats must list staking_tier_penalty")
	}
}
