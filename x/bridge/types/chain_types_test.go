package types

import "testing"

// TestAllChainTypes_Stable locks in the v2.24.0 ChainType set so a future
// addition or reordering is caught by review (the migration tooling and
// some external integrations depend on the exact ordering).
func TestAllChainTypes_Stable(t *testing.T) {
	want := []ChainType{
		ChainTypeIBC,
		ChainTypeEVM,
		ChainTypeSolana,
		ChainTypeTON,
		ChainTypeSui,
		ChainTypeAptos,
		ChainTypeBitcoin,
		ChainTypeNEAR,
		ChainTypeCardano,
		ChainTypePolkadot,
		ChainTypeTezos,
		ChainTypeTron,
		ChainTypeStarknet,
		ChainTypeXRPL,
		ChainTypeStellar,
		ChainTypeHedera,
		ChainTypeAlgorand,
	}
	got := AllChainTypes()
	if len(got) != len(want) {
		t.Fatalf("AllChainTypes() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllChainTypes()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsValidChainType(t *testing.T) {
	cases := []struct {
		in   ChainType
		want bool
	}{
		// Pre-existing
		{ChainTypeEVM, true},
		{ChainTypeIBC, true},
		{ChainTypeBitcoin, true},
		// New in v2.24.0
		{ChainTypeStarknet, true},
		{ChainTypeXRPL, true},
		{ChainTypeStellar, true},
		{ChainTypeHedera, true},
		{ChainTypeAlgorand, true},
		// Invalid
		{ChainType(""), false},
		{ChainType("unknown"), false},
		{ChainType("EVM"), false}, // case-sensitive
		{ChainType("ALGORAND"), false},
	}
	for _, c := range cases {
		if got := IsValidChainType(c.in); got != c.want {
			t.Errorf("IsValidChainType(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestNewChainTypes_StringValues — the wire-level string is the source of
// truth for cross-system compatibility (sidecar containers, indexers, the
// dashboard). Lock the values explicitly so a typo is caught at test time.
func TestNewChainTypes_StringValues(t *testing.T) {
	cases := []struct {
		ct   ChainType
		want string
	}{
		{ChainTypeStarknet, "starknet"},
		{ChainTypeXRPL, "xrpl"},
		{ChainTypeStellar, "stellar"},
		{ChainTypeHedera, "hedera"},
		{ChainTypeAlgorand, "algorand"},
	}
	for _, c := range cases {
		if string(c.ct) != c.want {
			t.Errorf("ChainType %q value = %q, want %q", c.ct, string(c.ct), c.want)
		}
	}
}

// TestChainType_NoCollisions — every ChainType string must be unique.
func TestChainType_NoCollisions(t *testing.T) {
	seen := map[ChainType]bool{}
	for _, ct := range AllChainTypes() {
		if seen[ct] {
			t.Errorf("duplicate ChainType: %q", ct)
		}
		seen[ct] = true
	}
}
