package types

import (
	"strings"
	"testing"
)

// TestAllFeatureIDs_Counts locks the v2.26.0 expectations:
// - 1 QCB bridge umbrella
// - 36 bridge_* (16 pre-existing + 20 new)
// - 10 validator_* (unchanged in v2.26.0; expanded in v2.27.0)
// Total = 47.
func TestAllFeatureIDs_Counts(t *testing.T) {
	bridges := AllBridgeFeatureIDs()
	if len(bridges) != 36 {
		t.Errorf("AllBridgeFeatureIDs count = %d, want 36", len(bridges))
	}
	validators := AllValidatorFeatureIDs()
	if len(validators) != 10 {
		t.Errorf("AllValidatorFeatureIDs count = %d, want 10", len(validators))
	}
	all := AllFeatureIDs()
	if len(all) != 1+36+10 {
		t.Errorf("AllFeatureIDs count = %d, want %d", len(all), 1+36+10)
	}
}

// TestAllBridgeFeatureIDs_NoDuplicates
func TestAllBridgeFeatureIDs_NoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, id := range AllBridgeFeatureIDs() {
		if seen[id] {
			t.Errorf("duplicate bridge feature: %q", id)
		}
		seen[id] = true
	}
}

// TestAllBridgeFeatureIDs_PrefixInvariant — every bridge feature must
// start with "bridge_". This is depended on by ChainFromFeature.
func TestAllBridgeFeatureIDs_PrefixInvariant(t *testing.T) {
	for _, id := range AllBridgeFeatureIDs() {
		if !strings.HasPrefix(id, "bridge_") {
			t.Errorf("bridge feature %q missing 'bridge_' prefix", id)
		}
	}
}

// TestAllValidatorFeatureIDs_PrefixInvariant
func TestAllValidatorFeatureIDs_PrefixInvariant(t *testing.T) {
	for _, id := range AllValidatorFeatureIDs() {
		if !strings.HasPrefix(id, "validator_") {
			t.Errorf("validator feature %q missing 'validator_' prefix", id)
		}
	}
}

// TestNewBridgeFeatureIDs_Present — every chain added in v2.25.0 has a
// matching bridge feature ID in v2.26.0.
func TestNewBridgeFeatureIDs_Present(t *testing.T) {
	want := []string{
		FeatureBridgeZKSyncEra, FeatureBridgeLinea, FeatureBridgeScroll,
		FeatureBridgeStarknet, FeatureBridgeBlast, FeatureBridgeMantle,
		FeatureBridgeHyperliquid, FeatureBridgeBerachain, FeatureBridgeSonic,
		FeatureBridgeMonad, FeatureBridgePlasma, FeatureBridgeFilecoin,
		FeatureBridgeCronos, FeatureBridgeKaia, FeatureBridgeSei,
		FeatureBridgeXRPL, FeatureBridgeStellar, FeatureBridgeHedera,
		FeatureBridgeAlgorand, FeatureBridgeInjective,
	}
	if len(want) != 20 {
		t.Fatalf("test bug: want should have 20 entries")
	}
	for _, id := range want {
		if !IsValidFeatureID(id) {
			t.Errorf("new feature %q is not registered in AllFeatureIDs", id)
		}
	}
}

func TestIsValidFeatureID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{FeatureQCBBridge, true},
		{FeatureBridgeStarknet, true},
		{FeatureBridgeXRPL, true},
		{FeatureBridgeInjective, true},
		{FeatureValidatorEthereum, true},
		{"", false},
		{"bridge_unknown", false},
		{"validator_starknet", false}, // v2.27.0 adds this
		{"BRIDGE_ETHEREUM", false},    // case-sensitive
	}
	for _, c := range cases {
		if got := IsValidFeatureID(c.in); got != c.want {
			t.Errorf("IsValidFeatureID(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestChainFromFeature(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{FeatureBridgeEthereum, "ethereum"},
		{FeatureBridgeStarknet, "starknet"},
		{FeatureBridgeZKSyncEra, "zksync_era"},
		{FeatureBridgeXRPL, "xrpl"},
		{FeatureValidatorSolana, "solana"},
		{FeatureQCBBridge, ""},          // not a chain-bound feature
		{"unknown", ""},
		{"bridge_", ""},                  // empty suffix
	}
	for _, c := range cases {
		if got := ChainFromFeature(c.in); got != c.want {
			t.Errorf("ChainFromFeature(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
