package pqc

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestSHAKE256Hash_OutputLength(t *testing.T) {
	data := []byte("qorechain test data")

	tests := []struct {
		name      string
		outputLen int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
		{"128 bytes", 128},
		{"1 byte", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SHAKE256Hash(data, tc.outputLen)
			if len(result) != tc.outputLen {
				t.Errorf("expected output length %d, got %d", tc.outputLen, len(result))
			}
		})
	}
}

func TestSHAKE256Hash_Determinism(t *testing.T) {
	data := []byte("determinism check")
	r1 := SHAKE256Hash(data, 32)
	r2 := SHAKE256Hash(data, 32)
	if !bytes.Equal(r1, r2) {
		t.Errorf("SHAKE256Hash not deterministic: %x != %x", r1, r2)
	}
}

func TestSHAKE256Hash_DifferentInputs(t *testing.T) {
	r1 := SHAKE256Hash([]byte("input A"), 32)
	r2 := SHAKE256Hash([]byte("input B"), 32)
	if bytes.Equal(r1, r2) {
		t.Error("SHAKE256Hash produced same output for different inputs")
	}
}

func TestSHAKE256Hash_EmptyInput(t *testing.T) {
	result := SHAKE256Hash([]byte{}, 32)
	if len(result) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(result))
	}
	// SHAKE-256 of empty input should be a known value.
	// SHAKE-256("", 32) = 46b9dd2b0ba88d13233b3feb743eeb243fcd52ea62b81b82b50c27646ed5762f
	expected, _ := hex.DecodeString("46b9dd2b0ba88d13233b3feb743eeb243fcd52ea62b81b82b50c27646ed5762f")
	if !bytes.Equal(result, expected) {
		t.Errorf("SHAKE256 empty input:\n  got:      %x\n  expected: %x", result, expected)
	}
}

func TestSHAKE256Hash32(t *testing.T) {
	data := []byte("hash32 test")
	result := SHAKE256Hash32(data)

	// Should match SHAKE256Hash with 32-byte output
	expected := SHAKE256Hash(data, 32)
	if !bytes.Equal(result[:], expected) {
		t.Errorf("SHAKE256Hash32 != SHAKE256Hash(_, 32):\n  Hash32: %x\n  Hash:   %x", result[:], expected)
	}
}

func TestSHAKE256ConcatHash_Determinism(t *testing.T) {
	left := SHAKE256Hash([]byte("left child"), 32)
	right := SHAKE256Hash([]byte("right child"), 32)

	r1 := SHAKE256ConcatHash(left, right)
	r2 := SHAKE256ConcatHash(left, right)
	if r1 != r2 {
		t.Errorf("ConcatHash not deterministic: %x != %x", r1, r2)
	}
}

func TestSHAKE256ConcatHash_OrderMatters(t *testing.T) {
	a := SHAKE256Hash([]byte("node A"), 32)
	b := SHAKE256Hash([]byte("node B"), 32)

	ab := SHAKE256ConcatHash(a, b)
	ba := SHAKE256ConcatHash(b, a)
	if ab == ba {
		t.Error("ConcatHash(a,b) == ConcatHash(b,a) — order should matter")
	}
}

func TestSHAKE256ConcatHash_DiffersFromIndividual(t *testing.T) {
	left := []byte("left")
	right := []byte("right")

	concat := SHAKE256ConcatHash(left, right)
	direct := SHAKE256Hash32(append(left, right...))

	// ConcatHash uses Write(left) + Write(right) which is equivalent to
	// hashing the concatenation, so these should be equal.
	if concat != direct {
		t.Errorf("ConcatHash differs from direct concat hash:\n  concat: %x\n  direct: %x", concat, direct)
	}
}

func TestSHAKE256DomainHash_Separation(t *testing.T) {
	data := []byte("same data")

	leafHash := SHAKE256DomainHash("leaf:", data)
	nodeHash := SHAKE256DomainHash("node:", data)

	if leafHash == nodeHash {
		t.Error("domain separation failed: leaf and node hashes are equal")
	}
}

func TestSHAKE256DomainHash_Determinism(t *testing.T) {
	data := []byte("deterministic")
	r1 := SHAKE256DomainHash("test:", data)
	r2 := SHAKE256DomainHash("test:", data)
	if r1 != r2 {
		t.Errorf("DomainHash not deterministic: %x != %x", r1, r2)
	}
}

func TestSHAKE256_KnownVector(t *testing.T) {
	// SHAKE-256 of "abc" (3 bytes) with 32-byte output.
	// Verified against golang.org/x/crypto/sha3 reference implementation.
	data := []byte("abc")
	result := SHAKE256Hash(data, 32)

	expected, _ := hex.DecodeString("483366601360a8771c6863080cc4114d8db44530f8f1e1ee4f94ea37e78b5739")
	if !bytes.Equal(result, expected) {
		t.Errorf("SHAKE-256 known vector mismatch:\n  got:      %x\n  expected: %x", result, expected)
	}
}
