package qorehash

import (
	"bytes"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/sha3"
)

// TestSum256MatchesShake confirms Sum256 is exactly SHAKE-256 read to 32 bytes.
func TestSum256MatchesShake(t *testing.T) {
	data := []byte("qorechain canonical hash")
	got := Sum256(data)

	h := sha3.NewShake256()
	h.Write(data)
	var want [32]byte
	h.Read(want[:])

	if got != want {
		t.Fatalf("Sum256 != SHAKE-256\n got %x\nwant %x", got, want)
	}
}

// TestSum256NotSha256 guards against accidentally wiring SHA-256 back in.
func TestSum256NotSha256(t *testing.T) {
	// Known SHAKE-256 (32-byte) digest of the empty string (FIPS 202 / NIST).
	want, _ := hex.DecodeString("46b9dd2b0ba88d13233b3feb743eeb243fcd52ea62b81b82b50c27646ed5762f")
	got := Sum256(nil)
	if !bytes.Equal(got[:], want) {
		t.Fatalf("SHAKE-256(\"\")[:32] mismatch\n got %x\nwant %x", got, want)
	}
}

// TestHasherMatchesSum256 confirms the streaming Hasher equals the one-shot Sum256.
func TestHasherMatchesSum256(t *testing.T) {
	a := []byte("hello ")
	b := []byte("world")

	h := New()
	h.Write(a)
	h.Write(b)
	streamed := h.Sum(nil)

	oneShot := Sum256(append(append([]byte{}, a...), b...))
	if !bytes.Equal(streamed, oneShot[:]) {
		t.Fatalf("streaming Hasher != Sum256\n got %x\nwant %x", streamed, oneShot[:])
	}
}

// TestHasherSumIdempotent confirms Sum does not disturb the running state.
func TestHasherSumIdempotent(t *testing.T) {
	h := New()
	h.Write([]byte("abc"))
	first := h.Sum(nil)
	second := h.Sum(nil)
	if !bytes.Equal(first, second) {
		t.Fatalf("Sum not idempotent: %x vs %x", first, second)
	}
	// Writing more after a Sum must still extend the same stream.
	h.Write([]byte("def"))
	combined := h.Sum(nil)
	want := Sum256([]byte("abcdef"))
	if !bytes.Equal(combined, want[:]) {
		t.Fatalf("post-Sum write diverged\n got %x\nwant %x", combined, want[:])
	}
}

// TestConcatHashSecondPreimage confirms length-prefix framing disambiguates
// the split point between the two operands.
func TestConcatHashSecondPreimage(t *testing.T) {
	// ("ab","c") and ("a","bc") must NOT collide thanks to length prefixing.
	x := ConcatHash([]byte("ab"), []byte("c"))
	y := ConcatHash([]byte("a"), []byte("bc"))
	if x == y {
		t.Fatal("ConcatHash collided on ambiguous split — length prefix missing")
	}
}

func TestSizes(t *testing.T) {
	if Size != 32 {
		t.Fatalf("Size = %d, want 32", Size)
	}
	if New().BlockSize() != 136 {
		t.Fatalf("BlockSize = %d, want 136", New().BlockSize())
	}
}
