package types

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
)

// TestDefaultParams_Validate verifies the v2.0.0 defaults are accepted.
func TestDefaultParams_Validate(t *testing.T) {
	if err := DefaultParams().Validate(); err != nil {
		t.Fatalf("DefaultParams should validate, got: %v", err)
	}
}

// TestParams_RewardShareMatchesBurnLightNodeShare locks in the v2.0.5
// invariant that the lightnode reward share is 3% — the same value the
// burn module's fee distribution allocates to light nodes. If these drift
// apart the chain will either over- or under-pay the light node pool.
func TestParams_RewardShareMatchesBurnLightNodeShare(t *testing.T) {
	p := DefaultParams()
	want := math.LegacyNewDecWithPrec(3, 2) // 0.03
	if !p.RewardShare.Equal(want) {
		t.Errorf("light node reward_share: got %s, want %s — must match burn.LightNodeShare",
			p.RewardShare, want)
	}
}

// TestParams_Validate_RejectsRewardShareOverOne — reward share is a fraction.
func TestParams_Validate_RejectsRewardShareOverOne(t *testing.T) {
	p := DefaultParams()
	p.RewardShare = math.LegacyNewDecWithPrec(150, 2) // 1.5
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for reward_share > 1.0")
	}
}

// TestParams_Validate_RejectsNegativeRewardShare
func TestParams_Validate_RejectsNegativeRewardShare(t *testing.T) {
	p := DefaultParams()
	p.RewardShare = math.LegacyNewDecWithPrec(-5, 2)
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for negative reward_share")
	}
}

// TestParams_Validate_RejectsZeroHeartbeatInterval prevents division-by-zero
// in the EndBlocker uptime calculation.
func TestParams_Validate_RejectsZeroHeartbeatInterval(t *testing.T) {
	p := DefaultParams()
	p.HeartbeatInterval = 0
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for heartbeat_interval == 0")
	}
	if !strings.Contains(err.Error(), "heartbeat_interval") {
		t.Errorf("expected heartbeat_interval error, got: %v", err)
	}
}

// TestParams_Validate_RejectsZeroMaxLightNodes prevents an empty registry
// from being indistinguishable from "no cap configured".
func TestParams_Validate_RejectsZeroMaxLightNodes(t *testing.T) {
	p := DefaultParams()
	p.MaxLightNodes = 0
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for max_light_nodes == 0")
	}
}

// TestParams_Validate_RejectsNegativeGracePeriod prevents underflow in
// the EndBlocker check (lastHeartbeat + interval + grace > height).
func TestParams_Validate_RejectsNegativeGracePeriod(t *testing.T) {
	p := DefaultParams()
	p.HeartbeatGracePeriod = -1
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for negative heartbeat_grace_period")
	}
}

// TestParams_Validate_RejectsUptimeOverOne — uptime ratio is bounded [0,1].
func TestParams_Validate_RejectsUptimeOverOne(t *testing.T) {
	p := DefaultParams()
	p.MinUptimeForRewards = math.LegacyNewDecWithPrec(150, 2)
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for min_uptime_for_rewards > 1.0")
	}
}
