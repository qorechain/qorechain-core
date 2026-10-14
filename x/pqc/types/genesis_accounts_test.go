package types

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QSR-2026-0354: genesis checks the PQC account records it imports, and
// still accepts what a live chain can export.
func TestGenesisValidatesAccounts(t *testing.T) {
	addr := sdk.AccAddress(bytes.Repeat([]byte{4}, 20)).String()
	other := sdk.AccAddress(bytes.Repeat([]byte{5}, 20)).String()
	ok := PQCAccountInfo{Address: addr, PublicKey: mldsaKey(7), AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeHybrid}
	cases := []struct {
		name     string
		accounts []PQCAccountInfo
		ok       bool
	}{
		{"valid", []PQCAccountInfo{ok}, true},
		{"classical_only without key (live diana shape)", []PQCAccountInfo{{Address: addr, KeyType: KeyTypeClassicalOnly, AlgorithmID: AlgorithmDilithium5}}, true},
		{"degenerate key (live diana shape)", []PQCAccountInfo{{Address: addr, PublicKey: make([]byte, 2592), AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeHybrid}}, true},
		{"retired hashes", []PQCAccountInfo{{Address: addr, PublicKey: mldsaKey(7), AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeHybrid, RetiredKeyHashes: [][]byte{RetiredKeyHash(mldsaKey(1))}}}, true},
		{"bad address", []PQCAccountInfo{{Address: "qor1nope", PublicKey: mldsaKey(7), AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeHybrid}}, false},
		{"duplicate", []PQCAccountInfo{ok, ok}, false},
		{"hybrid without key", []PQCAccountInfo{{Address: addr, AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeHybrid}}, false},
		{"unknown key type", []PQCAccountInfo{{Address: addr, PublicKey: mldsaKey(7), AlgorithmID: AlgorithmDilithium5, KeyType: "weird"}}, false},
		{"KEM as auth key", []PQCAccountInfo{{Address: addr, PublicKey: make([]byte, 1568), AlgorithmID: AlgorithmMLKEM1024, KeyType: KeyTypeHybrid}}, false},
		{"wrong length", []PQCAccountInfo{{Address: addr, PublicKey: []byte{1, 2}, AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeHybrid}}, false},
		{"bad retired hash", []PQCAccountInfo{{Address: other, PublicKey: mldsaKey(7), AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeHybrid, RetiredKeyHashes: [][]byte{{1}}}}, false},
	}
	for _, c := range cases {
		gs := DefaultGenesisState()
		gs.Accounts = c.accounts
		if err := gs.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: err=%v want ok=%v", c.name, err, c.ok)
		}
	}
}
