//go:build full

package pqc_test

import (
	"testing"

	qpqc "github.com/qorechain/qorechain-pqc/go"

	"github.com/qorechain/qorechain-core/x/pqc/ffi"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// TestQPQCSignVerifiesUnderFFI proves the client-side standards signer
// (qorechain-pqc/go, circl ML-DSA-87) produces signatures the chain's FFI
// verifier accepts — the exact cross-implementation boundary the hybrid cosign
// path relies on. If this holds, `recover-key` → `cosign` (qpqc.Sign) → on-chain
// FFI verify round-trips.
func TestQPQCSignVerifiesUnderFFI(t *testing.T) {
	pub, sec, err := qpqc.MLDSA87.Keygen()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("qorechain hybrid pqc interop check")
	sig, err := qpqc.MLDSA87.Sign(sec, msg)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := ffi.NewFFIClient().Verify(types.AlgorithmDilithium5, pub, msg, sig)
	if err != nil {
		t.Fatalf("FFI verify error: %v", err)
	}
	if !ok {
		t.Fatal("qorechain-pqc/go signature did NOT verify under the chain FFI — cross-impl mismatch")
	}
}

// TestQPQCKeygenFromSeedDeterministic confirms the seed→key derivation the
// recover-key command uses is reproducible (same seed → same pubkey).
func TestQPQCKeygenFromSeedDeterministic(t *testing.T) {
	seed := qpqc.Shake256([]byte("qorechain:pqc:v1|qor1test|word word word"), 32)
	p1, _, err := qpqc.MLDSA87.KeygenFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	p2, _, err := qpqc.MLDSA87.KeygenFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	if string(p1) != string(p2) {
		t.Fatal("KeygenFromSeed not deterministic")
	}
	// And a recovered key must also verify under the FFI.
	sig, err := qpqc.MLDSA87.Sign(mustSec(t, seed), []byte("m"))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := ffi.NewFFIClient().Verify(types.AlgorithmDilithium5, p1, []byte("m"), sig)
	if err != nil || !ok {
		t.Fatalf("recovered-key signature failed FFI verify (ok=%v err=%v)", ok, err)
	}
}

func mustSec(t *testing.T, seed []byte) []byte {
	_, sec, err := qpqc.MLDSA87.KeygenFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	return sec
}
