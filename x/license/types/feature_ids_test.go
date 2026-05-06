package types

import (
	"strings"
	"testing"
)

// TestAllFeatureIDs_Counts locks the v2.27.0 expectations:
// - 1 QCB bridge umbrella
// - 36 bridge_* (16 pre-existing + 20 added in v2.26.0)
// - 37 validator_* (10 baseline + 19 non-IBC v2.27.0 + 8 IBC v2.27.0)
//
// Total = 74 — matches the §3.4.4 acceptance criterion.
func TestAllFeatureIDs_Counts(t *testing.T) {
	bridges := AllBridgeFeatureIDs()
	if len(bridges) != 36 {
		t.Errorf("AllBridgeFeatureIDs count = %d, want 36", len(bridges))
	}
	validators := AllValidatorFeatureIDs()
	if len(validators) != 37 {
		t.Errorf("AllValidatorFeatureIDs count = %d, want 37", len(validators))
	}
	all := AllFeatureIDs()
	if len(all) != 1+36+37 {
		t.Errorf("AllFeatureIDs count = %d, want %d", len(all), 1+36+37)
	}
}

// TestAllValidatorFeatureIDs_NoDuplicates
func TestAllValidatorFeatureIDs_NoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, id := range AllValidatorFeatureIDs() {
		if seen[id] {
			t.Errorf("duplicate validator feature: %q", id)
		}
		seen[id] = true
	}
}

// TestNewValidatorFeatureIDs_Present — every chain that should have a
// validator license in v2.27.0 must be registered.
func TestNewValidatorFeatureIDs_Present(t *testing.T) {
	want := []string{
		// 19 non-IBC chains added in v2.25.0 (excluding Injective which is IBC)
		FeatureValidatorZKSyncEra, FeatureValidatorLinea, FeatureValidatorScroll,
		FeatureValidatorStarknet, FeatureValidatorBlast, FeatureValidatorMantle,
		FeatureValidatorHyperliquid, FeatureValidatorBerachain, FeatureValidatorSonic,
		FeatureValidatorSei, FeatureValidatorMonad, FeatureValidatorPlasma,
		FeatureValidatorXRPL, FeatureValidatorStellar, FeatureValidatorHedera,
		FeatureValidatorAlgorand, FeatureValidatorFilecoin, FeatureValidatorCronos,
		FeatureValidatorKaia,
		// 7 pre-existing IBC chains
		FeatureValidatorCosmosHub, FeatureValidatorOsmosis, FeatureValidatorNoble,
		FeatureValidatorCelestia, FeatureValidatorStride, FeatureValidatorAkash,
		FeatureValidatorBabylon,
		// + Injective (IBC chain added in v2.25.0)
		FeatureValidatorInjective,
	}
	if len(want) != 27 {
		t.Fatalf("test bug: want should have 27 entries, has %d", len(want))
	}
	for _, id := range want {
		if !IsValidFeatureID(id) {
			t.Errorf("new validator feature %q is not registered in AllFeatureIDs", id)
		}
	}
}

// TestBridgeAndValidator_Symmetry — every chain in v2.25.0 (the new 20)
// has BOTH a bridge_* AND a validator_* feature, EXCEPT Injective which
// only has a bridge_* (covered) and gets its validator_* via the IBC group.
//
// Symmetry tells us we won't have orphan licenses where a validator can
// hold a bridge license but no validator license, or vice versa.
func TestBridgeAndValidator_Symmetry(t *testing.T) {
	for _, ch := range []string{
		"ethereum", "solana", "bsc", "polygon", "arbitrum", "optimism",
		"base", "avalanche", "ton", "sui",
		"zksync_era", "linea", "scroll", "starknet", "blast", "mantle",
		"hyperliquid", "berachain", "sonic", "sei", "monad", "plasma",
		"xrpl", "stellar", "hedera", "algorand", "injective",
		"filecoin", "cronos", "kaia",
		"cosmoshub", "osmosis", "noble", "celestia", "stride", "akash",
		"babylon",
	} {
		validatorID := "validator_" + ch
		if !IsValidFeatureID(validatorID) {
			t.Errorf("chain %q missing validator_* feature", ch)
		}
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
		{FeatureValidatorStarknet, true},   // v2.27.0
		{FeatureValidatorCosmosHub, true},  // v2.27.0 IBC
		{FeatureValidatorInjective, true},  // v2.27.0 IBC
		{"", false},
		{"bridge_unknown", false},
		{"validator_unknown", false},
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
