package types

import (
	"testing"
)

func TestNewChainTypes(t *testing.T) {
	// Verify all 7 new chain type constants added in v1.2.0 exist and have the correct values.
	expected := map[string]ChainType{
		"aptos":    ChainTypeAptos,
		"bitcoin":  ChainTypeBitcoin,
		"near":     ChainTypeNEAR,
		"cardano":  ChainTypeCardano,
		"polkadot": ChainTypePolkadot,
		"tezos":    ChainTypeTezos,
		"tron":     ChainTypeTron,
	}

	for wantVal, got := range expected {
		if string(got) != wantVal {
			t.Errorf("ChainType mismatch: expected %q, got %q", wantVal, string(got))
		}
	}

	// Confirm they are distinct from the pre-existing types.
	preExisting := []ChainType{ChainTypeIBC, ChainTypeEVM, ChainTypeSolana, ChainTypeTON, ChainTypeSui}
	newTypes := []ChainType{ChainTypeAptos, ChainTypeBitcoin, ChainTypeNEAR, ChainTypeCardano, ChainTypePolkadot, ChainTypeTezos, ChainTypeTron}
	for _, n := range newTypes {
		for _, p := range preExisting {
			if n == p {
				t.Errorf("new chain type %q collides with pre-existing type %q", n, p)
			}
		}
	}
}

func TestDefaultChainConfigsCount(t *testing.T) {
	configs := DefaultChainConfigs()
	// v1.2.0 baseline was 17. v2.25.0 added 20 more for the cross-network
	// expansion (zkSync, Linea, Scroll, Starknet, Blast, Mantle,
	// Hyperliquid, Berachain, Sonic, Sei, Monad, Plasma, XRPL, Stellar,
	// Hedera, Algorand, Injective, Filecoin, Cronos, Kaia).
	if len(configs) != 37 {
		t.Fatalf("expected 37 chain configs, got %d", len(configs))
	}
}

func TestDefaultChainConfigsNewEntries(t *testing.T) {
	configs := DefaultChainConfigs()

	// Build a set of chain IDs present in the defaults.
	idSet := make(map[string]bool, len(configs))
	for _, c := range configs {
		idSet[c.ChainID] = true
	}

	// The 9 new chains added in v1.2.0 (optimism, base, plus the 7 new chain types).
	newChainIDs := []string{
		"optimism",
		"base",
		"aptos",
		"bitcoin",
		"near",
		"cardano",
		"polkadot",
		"tezos",
		"tron",
	}

	for _, id := range newChainIDs {
		if !idSet[id] {
			t.Errorf("expected chain %q in DefaultChainConfigs, but not found", id)
		}
	}
}
