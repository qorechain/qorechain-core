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

// TestRotationDualSigVerifiesUnderFFI mirrors rotatePQCKeyImpl's crypto path: it
// derives an OLD (legacy "bridge" derivation) and NEW (canonical "adapter")
// ML-DSA-87 key from the same mnemonic, has BOTH sign the exact
// RotationSignBytes the handler re-derives, and asserts both signatures verify
// under the chain FFI. This is the cross-impl boundary `tx pqc rotate-key` relies
// on, and proves a legacy→canonical rotation the handler will accept.
func TestRotationDualSigVerifiesUnderFFI(t *testing.T) {
	const (
		chainID  = "qorechain-diana"
		address  = "qor1wv0fvt5qzx7gllk9ckzv3u6ypceaqq8evuny0h"
		mnemonic = "test test test test test test test test test test test junk"
	)
	// OLD = bridge (mnemonic-only); NEW = adapter (address-bound).
	oldSeed := qpqc.Shake256([]byte(mnemonic), 32)
	newSeed := qpqc.Shake256([]byte("qorechain:pqc:v1|"+address+"|"+mnemonic), 32)
	oldPub, oldSec, err := qpqc.MLDSA87.KeygenFromSeed(oldSeed)
	if err != nil {
		t.Fatal(err)
	}
	newPub, newSec, err := qpqc.MLDSA87.KeygenFromSeed(newSeed)
	if err != nil {
		t.Fatal(err)
	}
	if string(oldPub) == string(newPub) {
		t.Fatal("bridge and adapter derivations collided — rotation would be a no-op")
	}

	signBytes := types.RotationSignBytes(chainID, uint32(types.AlgorithmDilithium5), address, oldPub, newPub)
	oldSig, err := qpqc.MLDSA87.Sign(oldSec, signBytes)
	if err != nil {
		t.Fatal(err)
	}
	newSig, err := qpqc.MLDSA87.Sign(newSec, signBytes)
	if err != nil {
		t.Fatal(err)
	}

	fc := ffi.NewFFIClient()
	if ok, err := fc.Verify(types.AlgorithmDilithium5, oldPub, signBytes, oldSig); err != nil || !ok {
		t.Fatalf("OLD-key rotation signature failed FFI verify (ok=%v err=%v)", ok, err)
	}
	if ok, err := fc.Verify(types.AlgorithmDilithium5, newPub, signBytes, newSig); err != nil || !ok {
		t.Fatalf("NEW-key rotation signature failed FFI verify (ok=%v err=%v)", ok, err)
	}
	// Cross-check: the OLD signature must NOT verify against the NEW key (the
	// handler's dual-sig check would otherwise be forgeable).
	if ok, _ := fc.Verify(types.AlgorithmDilithium5, newPub, signBytes, oldSig); ok {
		t.Fatal("OLD signature verified under NEW key — dual-sig binding is broken")
	}
}

func mustSec(t *testing.T, seed []byte) []byte {
	_, sec, err := qpqc.MLDSA87.KeygenFromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	return sec
}
