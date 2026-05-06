package types

import "testing"

// TestDefaultChainConfigs_Count locks the v2.25.0 chain count at 37
// (17 pre-existing + 20 added in v2.25.0).
func TestDefaultChainConfigs_Count(t *testing.T) {
	got := len(DefaultChainConfigs())
	const want = 37
	if got != want {
		t.Fatalf("DefaultChainConfigs() count = %d, want %d", got, want)
	}
}

// TestDefaultChainConfigs_NewChainsPresent verifies every chain added in
// v2.25.0 is in the default config list. Missing entries here would mean
// the chain is unsupported at boot, which would silently break the
// downstream license / sidecar wiring.
func TestDefaultChainConfigs_NewChainsPresent(t *testing.T) {
	wantNew := []string{
		"zksync_era", "linea", "scroll", "starknet", "blast", "mantle",
		"hyperliquid", "berachain", "sonic", "sei", "monad", "plasma",
		"xrpl", "stellar", "hedera", "algorand", "injective", "filecoin",
		"cronos", "kaia",
	}
	if len(wantNew) != 20 {
		t.Fatalf("test bug: wantNew should have 20 entries, has %d", len(wantNew))
	}
	got := DefaultChainConfigs()
	gotIDs := make(map[string]ChainConfig, len(got))
	for _, c := range got {
		gotIDs[c.ChainID] = c
	}
	for _, id := range wantNew {
		if _, ok := gotIDs[id]; !ok {
			t.Errorf("DefaultChainConfigs missing chain %q", id)
		}
	}
}

// TestDefaultChainConfigs_NoDuplicateChainIDs — defensive: prevents two
// configs from having the same ChainID, which would silently shadow the
// first one in any map-based lookup.
func TestDefaultChainConfigs_NoDuplicateChainIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range DefaultChainConfigs() {
		if seen[c.ChainID] {
			t.Errorf("duplicate chain_id: %q", c.ChainID)
		}
		seen[c.ChainID] = true
	}
}

// TestDefaultChainConfigs_ValidChainTypes — every config must have a
// ChainType that passes IsValidChainType. This is the integration point
// between v2.24.0's type list and v2.25.0's config list.
func TestDefaultChainConfigs_ValidChainTypes(t *testing.T) {
	for _, c := range DefaultChainConfigs() {
		if !IsValidChainType(c.ChainType) {
			t.Errorf("chain %q has invalid chain_type %q", c.ChainID, c.ChainType)
		}
	}
}

// TestDefaultChainConfigs_NewChainTypesUsed — the 5 ChainTypes added in
// v2.24.0 must appear in at least one config in v2.25.0 (otherwise they're
// unreachable dead code). Maps each new type to its expected chain_id.
func TestDefaultChainConfigs_NewChainTypesUsed(t *testing.T) {
	want := map[ChainType]string{
		ChainTypeStarknet: "starknet",
		ChainTypeXRPL:     "xrpl",
		ChainTypeStellar:  "stellar",
		ChainTypeHedera:   "hedera",
		ChainTypeAlgorand: "algorand",
	}
	got := map[ChainType]string{}
	for _, c := range DefaultChainConfigs() {
		if _, ok := want[c.ChainType]; ok {
			got[c.ChainType] = c.ChainID
		}
	}
	for ct, expectedID := range want {
		if got[ct] != expectedID {
			t.Errorf("ChainType %q: got chain_id %q, want %q", ct, got[ct], expectedID)
		}
	}
}

// TestDefaultChainConfigs_AllPending — every config defaults to Pending.
// Production deployment flips to Active only after the bridge contract
// address is set and the operator has confirmed the route.
func TestDefaultChainConfigs_AllPending(t *testing.T) {
	for _, c := range DefaultChainConfigs() {
		if c.Status != BridgeStatusPending {
			t.Errorf("chain %q: status = %q, want pending", c.ChainID, c.Status)
		}
	}
}
