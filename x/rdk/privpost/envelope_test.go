package privpost

import (
	"bytes"
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/cloudflare/circl/kem/mlkem/mlkem1024"
)

func member(t *testing.T, seed byte) ([]byte, []byte) {
	t.Helper()
	s := bytes.Repeat([]byte{seed}, mlkem1024.KeySeedSize)
	pk, sk := mlkem1024.NewKeyFromSeed(s)
	ek, _ := pk.MarshalBinary()
	exp, _ := sk.MarshalBinary()
	return ek, append(s, exp...)
}

func TestSealOpen(t *testing.T) {
	ekA, kA := member(t, 1)
	ekB, kB := member(t, 2)
	_, kC := member(t, 3)
	seedA, expB := kA[:mlkem1024.KeySeedSize], kB[mlkem1024.KeySeedSize:]
	recips := []Recipient{{"memberA", ekA}, {"memberB", ekB}}
	msg := []byte("private rollup transaction")
	env, err := Seal(rand.Reader, "chain", "priv-roll", 4, recips, msg)
	if err != nil {
		t.Fatal(err)
	}
	e, err := Parse(env)
	if err != nil || e.Epoch != 4 || len(e.Entries) != 2 || e.Entries[1].Fingerprint != Fingerprint(ekB) {
		t.Fatalf("parse %+v %v", e, err)
	}
	// Both members open it, with the seed and with the expanded key.
	for _, c := range []struct {
		who string
		key []byte
	}{{"memberA", seedA}, {"memberB", expB}} {
		pt, err := Open(env, "chain", "priv-roll", c.who, c.key)
		if err != nil || !bytes.Equal(pt, msg) {
			t.Fatalf("%s: %q %v", c.who, pt, err)
		}
	}
	// A non-member, the wrong key, another rollup or chain, or a changed
	// byte all fail.
	if _, err := Open(env, "chain", "priv-roll", "memberC", kC[:64]); !errors.Is(err, ErrNotMember) {
		t.Fatalf("non-member: %v", err)
	}
	if _, err := Open(env, "chain", "priv-roll", "memberA", kC[:64]); !errors.Is(err, ErrOpen) {
		t.Fatalf("wrong key: %v", err)
	}
	if _, err := Open(env, "chain", "other-roll", "memberA", seedA); !errors.Is(err, ErrOpen) {
		t.Fatalf("other rollup: %v", err)
	}
	tampered := append([]byte(nil), env...)
	tampered[len(tampered)-1] ^= 1
	if _, err := Open(tampered, "chain", "priv-roll", "memberA", seedA); !errors.Is(err, ErrOpen) {
		t.Fatalf("tampered payload: %v", err)
	}
	hdr := append([]byte(nil), env...)
	hdr[6] ^= 1 // epoch is authenticated
	if _, err := Open(hdr, "chain", "priv-roll", "memberA", seedA); !errors.Is(err, ErrOpen) {
		t.Fatalf("tampered header: %v", err)
	}
	// Size checks.
	for _, bad := range [][]byte{nil, []byte("QPRV"), env[:len(env)-len(msg)-1], append([]byte("XPRV"), env[4:]...)} {
		if _, err := Parse(bad); err == nil {
			t.Fatalf("parsed a malformed envelope of %d bytes", len(bad))
		}
	}
}

// A poster cannot give two members different content: the content key is
// committed in the header, so a member that unwraps a different key refuses.
func TestKeyCommitmentStopsEquivocation(t *testing.T) {
	ekA, kA := member(t, 1)
	ekB, kB := member(t, 2)
	envA, _ := Seal(rand.Reader, "chain", "r1x", 1, []Recipient{{"memberA", ekA}, {"memberB", ekB}}, []byte("to A"))
	envB, _ := Seal(rand.Reader, "chain", "r1x", 1, []Recipient{{"memberA", ekA}, {"memberB", ekB}}, []byte("to B"))
	a, _ := Parse(envA)
	b, _ := Parse(envB)
	// Splice B's entry for memberB into A's envelope.
	mixed := append([]byte(nil), envA...)
	off := bytes.Index(mixed, a.Entries[1].KEMCiphertext)
	copy(mixed[off:], b.Entries[1].KEMCiphertext)
	copy(mixed[off+KEMCiphertextSize:], b.Entries[1].WrappedKey)
	// memberB unwraps a content key that is not the committed one and
	// refuses; memberA refuses too, because the header is authenticated.
	_, err := Open(mixed, "chain", "r1x", "memberB", kB[:64])
	if !errors.Is(err, ErrOpen) || !strings.Contains(err.Error(), "commitment") {
		t.Fatalf("memberB: %v", err)
	}
	if _, err := Open(mixed, "chain", "r1x", "memberA", kA[:64]); !errors.Is(err, ErrOpen) {
		t.Fatalf("memberA opened a spliced envelope: %v", err)
	}
}
