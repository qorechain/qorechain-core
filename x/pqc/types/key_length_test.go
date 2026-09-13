package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func regSender() string { return sdk.AccAddress([]byte("a-registering-accoun")).String() }

// Registering a key of the wrong length is unrecoverable. Under the
// post-quantum-required posture the account can then never produce a valid
// signature for anything, and both rotate and migrate verify against the key
// already on file, so the repair needs the very capability the mistake removed.
// Only non-emptiness was checked, which let one transaction do it.
func TestRegistrationRefusesAKeyOfTheWrongLength(t *testing.T) {
	want := int(DefaultDilithium5Info().PublicKeySize)

	for _, n := range []int{1, 32, want - 1, want + 1, want * 2} {
		msg := &MsgRegisterPQCKey{
			Sender:          regSender(),
			DilithiumPubkey: make([]byte, n),
			KeyType:         KeyTypeHybrid,
		}
		if err := msg.ValidateBasic(); err == nil {
			t.Fatalf("a %d-byte key was accepted; %d is the only length that can sign", n, want)
		}
	}

	good := &MsgRegisterPQCKey{
		Sender:          regSender(),
		DilithiumPubkey: make([]byte, want),
		KeyType:         KeyTypeHybrid,
	}
	if err := good.ValidateBasic(); err != nil {
		t.Fatalf("a correctly sized key was refused: %v", err)
	}
}

// The same rule on the V2 message, which carries the algorithm explicitly.
func TestRegistrationV2RefusesAKeyOfTheWrongLength(t *testing.T) {
	want := int(DefaultDilithium5Info().PublicKeySize)

	bad := &MsgRegisterPQCKeyV2{
		Sender:      regSender(),
		AlgorithmID: AlgorithmDilithium5,
		PublicKey:   make([]byte, want-1),
		KeyType:     KeyTypeHybrid,
	}
	if err := bad.ValidateBasic(); err == nil {
		t.Fatal("a short key was accepted for a known algorithm")
	}

	good := &MsgRegisterPQCKeyV2{
		Sender:      regSender(),
		AlgorithmID: AlgorithmDilithium5,
		PublicKey:   make([]byte, want),
		KeyType:     KeyTypeHybrid,
	}
	if err := good.ValidateBasic(); err != nil {
		t.Fatalf("a correctly sized key was refused: %v", err)
	}
}

// An algorithm added by governance after this binary shipped has no size known
// here, so the stateless check must not guess. Refusing it would block a
// legitimately added algorithm; the keeper checks it against the on-chain registry,
// which is where the answer actually lives.
func TestUnknownAlgorithmIsLeftToTheKeeper(t *testing.T) {
	if _, known := ExpectedPublicKeySize(AlgorithmID(99)); known {
		t.Fatal("an algorithm this binary does not ship reported a known key size")
	}

	msg := &MsgRegisterPQCKeyV2{
		Sender:      regSender(),
		AlgorithmID: AlgorithmID(99),
		PublicKey:   make([]byte, 7),
		KeyType:     KeyTypeHybrid,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("a governance-added algorithm was refused by the stateless check, "+
			"which cannot know its size: %v", err)
	}
}

// The sizes the check uses have to be the real ones. A wrong constant here would
// refuse every legitimate key, which is an outage rather than a defence.
func TestKnownKeySizesAreTheRealOnes(t *testing.T) {
	if got, _ := ExpectedPublicKeySize(AlgorithmDilithium5); got != 2592 {
		t.Fatalf("ML-DSA-87 public key size recorded as %d, want 2592", got)
	}
	if got, known := ExpectedPublicKeySize(AlgorithmMLKEM1024); !known || got == 0 {
		t.Fatalf("ML-KEM-1024 size is %d, known=%v", got, known)
	}
}
