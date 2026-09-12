package types

import "testing"

// ML-KEM is a key-encapsulation mechanism with no signing operation. Registering
// it as an authentication key leaves the account unable to sign and unable to
// rotate out, because rotation requires a valid signature under the key on file.
// Reported and reproduced on the public testnet by Akira Nakao (QSR-2026-0193).
func TestRegisterPQCKeyV2_RejectsKEMAlgorithm(t *testing.T) {
	msg := &MsgRegisterPQCKeyV2{
		Sender:      "qor1kndlxryvnkk3fz6kf7zzg9rw0yzch9cu7tj8us",
		AlgorithmID: AlgorithmMLKEM1024,
		PublicKey:   make([]byte, 1568), // a correctly sized ML-KEM key
		KeyType:     KeyTypePQCOnly,
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("a correctly sized ML-KEM key was accepted as an authentication key")
	}
}

func TestRegisterPQCKeyV2_AcceptsSignatureAlgorithm(t *testing.T) {
	msg := &MsgRegisterPQCKeyV2{
		Sender:      "qor1kndlxryvnkk3fz6kf7zzg9rw0yzch9cu7tj8us",
		AlgorithmID: AlgorithmDilithium5,
		PublicKey:   make([]byte, 2592),
		KeyType:     KeyTypePQCOnly,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Fatalf("a valid Dilithium-5 registration was rejected: %v", err)
	}
}
