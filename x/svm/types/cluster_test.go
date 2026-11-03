package types

import (
	"crypto/sha256"
	"strings"
	"testing"
)

// The identifiers below are what wallets will store to recognise each network.
// They must never change for a network: a change would make every wallet that
// saved the old value treat the network as a different cluster. If this test
// fails, the derivation changed - revert it, do not update the constants.
const (
	pinnedMainnetChainID     = "qorechain-vladi"
	pinnedMainnetGenesisHash = "4LQ9znKk8J3C8YBz2unNdqRCC35ZXvSiFN9YGJNwu3er"
	pinnedMainnetCAIP2       = "solana:4LQ9znKk8J3C8YBz2unNdqRCC35ZXvSi"

	pinnedDianaChainID     = "qorechain-diana"
	pinnedDianaGenesisHash = "6Z9pa53SwiprqiAoNCqqUdj5c3C4P1zRw31gQVnF4Czj"
	pinnedDianaCAIP2       = "solana:6Z9pa53SwiprqiAoNCqqUdj5c3C4P1zR"
)

func TestClusterGenesisHashPinned(t *testing.T) {
	for _, tc := range []struct{ chainID, hash, caip2 string }{
		{pinnedMainnetChainID, pinnedMainnetGenesisHash, pinnedMainnetCAIP2},
		{pinnedDianaChainID, pinnedDianaGenesisHash, pinnedDianaCAIP2},
	} {
		got, err := ClusterGenesisHashBase58(tc.chainID)
		if err != nil {
			t.Fatalf("%s: %v", tc.chainID, err)
		}
		if got != tc.hash {
			t.Fatalf("%s genesis hash = %s, pinned %s", tc.chainID, got, tc.hash)
		}
		caip2, err := ClusterCAIP2ChainID(tc.chainID)
		if err != nil {
			t.Fatalf("%s: %v", tc.chainID, err)
		}
		if caip2 != tc.caip2 {
			t.Fatalf("%s CAIP-2 = %s, pinned %s", tc.chainID, caip2, tc.caip2)
		}
	}
}

func TestClusterGenesisHashDeterministicAndDomainSeparated(t *testing.T) {
	a, err := ClusterGenesisHash("qorechain-vladi")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		b, _ := ClusterGenesisHash("qorechain-vladi")
		if a != b {
			t.Fatal("genesis hash is not deterministic")
		}
	}
	// Exact derivation: SHA-256(domain || chain id).
	if want := sha256.Sum256([]byte(ClusterGenesisHashDomain + "qorechain-vladi")); a != want {
		t.Fatal("genesis hash does not follow SHA-256(domain || chain id)")
	}
	// Domain separated: not the bare hash of the chain id.
	if bare := sha256.Sum256([]byte("qorechain-vladi")); a == bare {
		t.Fatal("genesis hash must be domain separated")
	}
	d, _ := ClusterGenesisHash("qorechain-diana")
	if a == d {
		t.Fatal("different chain ids must give different genesis hashes")
	}
}

func TestClusterGenesisHashRoundTripsAsBase58(t *testing.T) {
	for _, id := range []string{"qorechain-vladi", "qorechain-diana", "qorechain-devnet", "x"} {
		raw, err := ClusterGenesisHash(id)
		if err != nil {
			t.Fatal(err)
		}
		s, _ := ClusterGenesisHashBase58(id)
		back, err := Base58Decode(s)
		if err != nil {
			t.Fatalf("%s: base58 does not decode: %v", id, err)
		}
		if back != raw {
			t.Fatalf("%s: base58 round trip mismatch", id)
		}
		caip2, _ := ClusterCAIP2ChainID(id)
		if !strings.HasPrefix(caip2, "solana:") || len(caip2) != len("solana:")+32 {
			t.Fatalf("%s: bad CAIP-2 chain id %q", id, caip2)
		}
		if caip2[len("solana:"):] != s[:32] {
			t.Fatalf("%s: CAIP-2 reference is not the first 32 chars of the genesis hash", id)
		}
	}
}

func TestClusterGenesisHashRejectsEmptyChainID(t *testing.T) {
	if _, err := ClusterGenesisHash(""); err == nil {
		t.Fatal("empty chain id must be refused")
	}
	if _, err := ClusterGenesisHashBase58(""); err == nil {
		t.Fatal("empty chain id must be refused")
	}
	if _, err := ClusterCAIP2ChainID(""); err == nil {
		t.Fatal("empty chain id must be refused")
	}
}
