package types

import (
	"bytes"
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func mldsaKey(fill byte) []byte { return bytes.Repeat([]byte{fill}, 2592) }

// A key whose t1 is all zero can be forged by anyone (QSR-2026-0538); rho
// alone does not matter.
func TestIsDegeneratePublicKey(t *testing.T) {
	zeroT1 := make([]byte, 2592)
	zeroT1[0] = 0x80 // the reporter's second key: a bit inside rho only
	oneBitT1 := make([]byte, 2592)
	oneBitT1[2591] = 1
	cases := []struct {
		name string
		algo AlgorithmID
		pub  []byte
		want bool
	}{
		{"all zero", AlgorithmDilithium5, make([]byte, 2592), true},
		{"rho set, t1 zero", AlgorithmDilithium5, zeroT1, true},
		{"t1 has a bit", AlgorithmDilithium5, oneBitT1, false},
		{"real-looking", AlgorithmDilithium5, mldsaKey(0xaa), false},
		{"wrong length", AlgorithmDilithium5, make([]byte, 100), false},
		{"unknown algorithm", AlgorithmID(99), make([]byte, 2592), false},
	}
	for _, c := range cases {
		if got := IsDegeneratePublicKey(c.algo, c.pub); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestValidateRegisteredKey(t *testing.T) {
	cases := []struct {
		name    string
		keyType string
		pub     []byte
		ok      bool
	}{
		{"hybrid key", KeyTypeHybrid, mldsaKey(0xaa), true},
		{"pqc_only key", KeyTypePQCOnly, mldsaKey(0x11), true},
		{"classical_only empty", KeyTypeClassicalOnly, nil, false},
		{"classical_only with a key", KeyTypeClassicalOnly, mldsaKey(0xaa), false},
		{"hybrid empty", KeyTypeHybrid, nil, false},
		{"hybrid t1 zero", KeyTypeHybrid, make([]byte, 2592), false},
	}
	for _, c := range cases {
		err := ValidateRegisteredKey(c.keyType, AlgorithmDilithium5, c.pub)
		if (err == nil) != c.ok {
			t.Errorf("%s: err=%v want ok=%v", c.name, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrWeakPQCKey) {
			t.Errorf("%s: %v is not ErrWeakPQCKey", c.name, err)
		}
	}
}

func TestIsRepairableRecord(t *testing.T) {
	if !IsRepairableRecord(PQCAccountInfo{KeyType: KeyTypeClassicalOnly}) {
		t.Error("an empty classical_only record must be repairable")
	}
	if !IsRepairableRecord(PQCAccountInfo{KeyType: KeyTypeHybrid}) {
		t.Error("a record without a key must be repairable")
	}
	if IsRepairableRecord(PQCAccountInfo{KeyType: KeyTypeHybrid, PublicKey: mldsaKey(1)}) {
		t.Error("a record with a usable key must not be replaceable")
	}
}

func TestRetiredKeys(t *testing.T) {
	var a PQCAccountInfo
	a.RetireKey(mldsaKey(1))
	a.RetireKey(mldsaKey(1))
	a.RetireKey(nil)
	if len(a.RetiredKeyHashes) != 1 {
		t.Fatalf("got %d retired hashes, want 1", len(a.RetiredKeyHashes))
	}
	if !a.HasRetiredKey(mldsaKey(1)) || a.HasRetiredKey(mldsaKey(2)) {
		t.Fatal("retired-key lookup is wrong")
	}
}

// The proof-of-possession bytes bind chain, algorithm, account and key.
func TestProofOfPossessionSignBytesBindEverything(t *testing.T) {
	base := ProofOfPossessionSignBytes("c", 1, "a", []byte{1})
	for _, other := range [][]byte{
		ProofOfPossessionSignBytes("d", 1, "a", []byte{1}),
		ProofOfPossessionSignBytes("c", 2, "a", []byte{1}),
		ProofOfPossessionSignBytes("c", 1, "b", []byte{1}),
		ProofOfPossessionSignBytes("c", 1, "a", []byte{2}),
	} {
		if bytes.Equal(base, other) {
			t.Fatalf("%q does not differ from %q", other, base)
		}
	}
}

func TestMempoolKeyCheck(t *testing.T) {
	sender := sdk.AccAddress(bytes.Repeat([]byte{3}, 20)).String()
	cases := []struct {
		name string
		msg  sdk.Msg
		ok   bool
	}{
		{"register hybrid", &MsgRegisterPQCKey{Sender: sender, DilithiumPubkey: mldsaKey(9), KeyType: KeyTypeHybrid}, true},
		{"register classical_only", &MsgRegisterPQCKey{Sender: sender, KeyType: KeyTypeClassicalOnly}, false},
		{"register v2 classical_only", &MsgRegisterPQCKeyV2{Sender: sender, AlgorithmID: AlgorithmDilithium5, KeyType: KeyTypeClassicalOnly}, false},
		{"register v2 t1 zero", &MsgRegisterPQCKeyV2{Sender: sender, AlgorithmID: AlgorithmDilithium5, PublicKey: make([]byte, 2592), KeyType: KeyTypeHybrid}, false},
		{"rotate ok", &MsgRotatePQCKey{Sender: sender, OldPublicKey: mldsaKey(1), NewPublicKey: mldsaKey(2)}, true},
		{"rotate size mismatch", &MsgRotatePQCKey{Sender: sender, OldPublicKey: mldsaKey(1), NewPublicKey: []byte{2}}, false},
		{"rotate to t1 zero", &MsgRotatePQCKey{Sender: sender, OldPublicKey: mldsaKey(1), NewPublicKey: make([]byte, 2592)}, false},
		{"migrate to t1 zero", &MsgMigratePQCKey{Sender: sender, NewAlgorithmID: AlgorithmDilithium5, NewPublicKey: make([]byte, 2592)}, false},
		{"other message", &MsgCloseEVMWindow{}, true},
	}
	for _, c := range cases {
		if err := MempoolKeyCheck(c.msg); (err == nil) != c.ok {
			t.Errorf("%s: err=%v want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestPQCKeyManagementMsgTypeURLs(t *testing.T) {
	got := PQCKeyManagementMsgTypeURLs()
	want := []string{
		"/qorechain.pqc.v1.MsgRegisterPQCKey",
		"/qorechain.pqc.v1.MsgRegisterPQCKeyV2",
		"/qorechain.pqc.v1.MsgRotatePQCKey",
		"/qorechain.pqc.v1.MsgMigratePQCKey",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %s want %s", got[i], want[i])
		}
	}
}

// Every v3.2.3 key-integrity feature is in this area's list, once, and is not
// already released by an earlier release list.
func TestV323PqcIntegrityFeatures(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range V323PqcIntegrityFeatures {
		if f == "" || seen[f] {
			t.Fatalf("empty or duplicate feature %q", f)
		}
		seen[f] = true
	}
	if len(seen) != 5 {
		t.Fatalf("got %d features, want 5", len(seen))
	}
	for _, f := range concatFeatures(ReleaseV3_2_0ProtocolFeatures, ReleaseV3_2_2ProtocolFeatures) {
		if seen[f] {
			t.Fatalf("%q was already released", f)
		}
	}
}
