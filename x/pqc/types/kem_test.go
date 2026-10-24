package types

import (
	"bytes"
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	qpqc "github.com/qorechain/qorechain-pqc/go"
)

const kemTestSender = "qor1kndlxryvnkk3fz6kf7zzg9rw0yzch9cu7tj8us"

func mlkemPub(t *testing.T) []byte {
	t.Helper()
	pub, _, err := qpqc.MLKEM1024.Keygen()
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

// A key from a conforming ML-KEM-1024 implementation passes, and a sender can
// encapsulate to exactly what passed.
func TestMLKEM1024KeyCheckAcceptsARealKey(t *testing.T) {
	for i := 0; i < 8; i++ {
		pub := mlkemPub(t)
		if err := ValidateMLKEM1024EncapsulationKey(pub); err != nil {
			t.Fatalf("a real ML-KEM-1024 key was refused: %v", err)
		}
		if _, _, err := qpqc.MLKEM1024.Encapsulate(pub); err != nil {
			t.Fatalf("encapsulation to an accepted key failed: %v", err)
		}
	}
}

// FIPS 203 section 7.2: a coefficient that is not reduced modulo q fails the
// encapsulation key check, in either half of a 3-byte group.
func TestMLKEM1024KeyCheckRefusesUnreducedCoefficients(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func([]byte)
	}{
		// first coefficient of the group = 0xfff
		{"low", func(p []byte) { p[0] = 0xff; p[1] |= 0x0f }},
		// second coefficient of the group = 3329 exactly (0xd01)
		{"high", func(p []byte) { p[3*7+1] = (p[3*7+1] & 0x0f) | 0x10; p[3*7+2] = 0xd0 }},
		// last group of t_hat
		{"last", func(p []byte) { p[mlkem1024THatBytes-1] = 0xff }},
	} {
		pub := mlkemPub(t)
		tc.edit(pub)
		if err := ValidateMLKEM1024EncapsulationKey(pub); !errors.Is(err, ErrInvalidKEMKey) {
			t.Fatalf("%s: an unreduced coefficient was accepted (err %v)", tc.name, err)
		}
	}
	// rho is a seed, not coefficients: any value is fine.
	pub := mlkemPub(t)
	for i := mlkem1024THatBytes; i < len(pub); i++ {
		pub[i] = 0xff
	}
	if err := ValidateMLKEM1024EncapsulationKey(pub); err != nil {
		t.Fatalf("a key with an all-0xff rho was refused: %v", err)
	}
}

func TestMLKEM1024KeyCheckRefusesZeroAndWrongSize(t *testing.T) {
	zero := make([]byte, 1568)
	zero[mlkem1024THatBytes] = 7 // any rho
	if err := ValidateMLKEM1024EncapsulationKey(zero); !errors.Is(err, ErrInvalidKEMKey) {
		t.Fatalf("a key with an all-zero t_hat was accepted: %v", err)
	}
	for _, n := range []int{0, 1567, 1569, 2592} {
		if err := ValidateMLKEM1024EncapsulationKey(make([]byte, n)); !errors.Is(err, ErrInvalidKEMKey) {
			t.Fatalf("a %d-byte key was accepted: %v", n, err)
		}
	}
}

func TestRegisterKEMKeyValidateBasic(t *testing.T) {
	good := &MsgRegisterKEMKey{Sender: kemTestSender, AlgorithmID: AlgorithmMLKEM1024, PublicKey: mlkemPub(t)}
	if err := good.ValidateBasic(); err != nil {
		t.Fatalf("a valid registration was refused: %v", err)
	}
	for name, msg := range map[string]*MsgRegisterKEMKey{
		"bad sender":          {Sender: "qor1nope", AlgorithmID: AlgorithmMLKEM1024, PublicKey: good.PublicKey},
		"no algorithm":        {Sender: kemTestSender, PublicKey: good.PublicKey},
		"signature algorithm": {Sender: kemTestSender, AlgorithmID: AlgorithmDilithium5, PublicKey: make([]byte, 2592)},
		"no key":              {Sender: kemTestSender, AlgorithmID: AlgorithmMLKEM1024},
		"short key":           {Sender: kemTestSender, AlgorithmID: AlgorithmMLKEM1024, PublicKey: good.PublicKey[:1000]},
		"oversized unknown":   {Sender: kemTestSender, AlgorithmID: 9, PublicKey: make([]byte, MaxKEMPublicKeySize+1)},
	} {
		if err := msg.ValidateBasic(); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
	// An algorithm governance adds later is left to the keeper's registry.
	later := &MsgRegisterKEMKey{Sender: kemTestSender, AlgorithmID: 9, PublicKey: make([]byte, 1184)}
	if err := later.ValidateBasic(); err != nil {
		t.Fatalf("a key of an algorithm unknown to this binary was refused statelessly: %v", err)
	}
	if err := (&MsgRemoveKEMKey{Sender: kemTestSender}).ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	if err := (&MsgRemoveKEMKey{Sender: "x"}).ValidateBasic(); err == nil {
		t.Fatal("a removal with a bad sender was accepted")
	}
}

// The KEM key messages are account key maintenance: refused inside authz with
// the other PQC key messages.
func TestKEMKeyMessagesAreKeyManagement(t *testing.T) {
	urls := map[string]bool{}
	for _, u := range PQCKeyManagementMsgTypeURLs() {
		urls[u] = true
	}
	for _, m := range []sdk.Msg{&MsgRegisterKEMKey{}, &MsgRemoveKEMKey{}} {
		if !urls[sdk.MsgTypeURL(m)] {
			t.Fatalf("%s is not in the authz-refused key management list", sdk.MsgTypeURL(m))
		}
		if !IsKEMKeyMsg(m) {
			t.Fatalf("%s is not recognised as a KEM key message", sdk.MsgTypeURL(m))
		}
	}
	if IsKEMKeyMsg(&MsgRegisterPQCKeyV2{}) {
		t.Fatal("a signature key registration counted as a KEM key message")
	}
}

// QSR-2026-0607, the reporter's case: a rotation whose new key has another
// length than the old one. ValidateBasic still accepts it (it runs in block
// execution, so it keeps its historical answer); the switch-gated rule refuses
// it from FeaturePQCRotateKeySizeCheck on, and only then.
func TestRotationKeySizeRuleWaitsForTheFeature(t *testing.T) {
	msg := &MsgRotatePQCKey{
		Sender:       kemTestSender,
		OldPublicKey: bytes.Repeat([]byte{1}, 2592),
		NewPublicKey: bytes.Repeat([]byte{2}, 100),
		OldSignature: []byte{1},
		NewSignature: []byte{1},
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("ValidateBasic changed its answer for a historical rotation: %v", err)
	}
	off := func(string) bool { return false }
	on := func(f string) bool { return f == FeaturePQCRotateKeySizeCheck }
	if err := FeatureGatedValidateBasic(msg, off); err != nil {
		t.Fatalf("below the switch the rotation was refused: %v", err)
	}
	if err := FeatureGatedValidateBasic(msg, on); !errors.Is(err, ErrInvalidKeyLength) {
		t.Fatalf("from the switch a mis-sized rotation was not refused: %v", err)
	}
	msg.NewPublicKey = bytes.Repeat([]byte{2}, 2592)
	if err := FeatureGatedValidateBasic(msg, on); err != nil {
		t.Fatalf("a same-size rotation was refused: %v", err)
	}
	// Other messages are untouched by the gated rules.
	if err := FeatureGatedValidateBasic(&MsgRegisterKEMKey{}, on); err != nil {
		t.Fatal(err)
	}
}

func TestPQCKEMFeaturesAreReleased(t *testing.T) {
	released := map[string]bool{}
	for _, f := range ReleaseV3_2_5ProtocolFeatures {
		released[f] = true
	}
	for _, f := range []string{FeaturePQCKEMRegistry, FeaturePQCRotateKeySizeCheck} {
		if !released[f] || !KnownProtocolFeature(f) {
			t.Fatalf("%s is not in the v3.2.5 release set", f)
		}
	}
}

func TestGenesisKEMKeys(t *testing.T) {
	addr := sdk.AccAddress(bytes.Repeat([]byte{0x61}, 20)).String()
	gs := DefaultGenesisState()
	gs.KEMKeys = []KEMKeyInfo{{Address: addr, AlgorithmID: AlgorithmMLKEM1024, PublicKey: mlkemPub(t), RegisteredHeight: 5}}
	if err := gs.Validate(); err != nil {
		t.Fatalf("a valid genesis KEM key was refused: %v", err)
	}
	dup := *gs
	dup.KEMKeys = append(append([]KEMKeyInfo{}, gs.KEMKeys...), gs.KEMKeys[0])
	if err := dup.Validate(); err == nil {
		t.Fatal("a duplicate genesis KEM key was accepted")
	}
	sig := *gs
	sig.KEMKeys = []KEMKeyInfo{{Address: addr, AlgorithmID: AlgorithmDilithium5, PublicKey: make([]byte, 2592)}}
	if err := sig.Validate(); err == nil {
		t.Fatal("a genesis KEM key under a signature algorithm was accepted")
	}
	bad := *gs
	bad.KEMKeys = []KEMKeyInfo{{Address: addr, AlgorithmID: AlgorithmMLKEM1024, PublicKey: make([]byte, 1568)}}
	if err := bad.Validate(); err == nil {
		t.Fatal("a genesis KEM key that fails the key check was accepted")
	}
}
