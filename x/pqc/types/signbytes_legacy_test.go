package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

// The legacy forms are what every block below the release switch verifies, so
// they are pinned byte for byte against the expressions the old binary used.
func TestLegacySignBytesAreTheOldExpressions(t *testing.T) {
	b0, auth := []byte("body-without-ext"), []byte("auth-info")
	want := make([]byte, 4+len(b0)+4+len(auth))
	binary.BigEndian.PutUint32(want[0:4], uint32(len(b0)))
	copy(want[4:], b0)
	binary.BigEndian.PutUint32(want[4+len(b0):], uint32(len(auth)))
	copy(want[8+len(b0):], auth)
	if got := HybridSignBytesLegacy(b0, auth); !bytes.Equal(got, want) {
		t.Fatalf("hybrid legacy form changed: %x vs %x", got, want)
	}
	if bytes.Equal(HybridSignBytesLegacy(b0, auth), HybridSignBytes("qorechain-vladi", b0, auth)) {
		t.Fatal("legacy and v2 must differ")
	}
	wantM := []byte(fmt.Sprintf("qorechain-key-migration:chain=%s:from=%d:to=%d:account=%s:height=%d", "qorechain-vladi", 1, 2, "qor1abc", 77))
	if got := MigrationSignBytesLegacy("qorechain-vladi", "qor1abc", 1, 2, 77); !bytes.Equal(got, wantM) {
		t.Fatalf("migration legacy form changed: %q", got)
	}
}

// The client-side choice of form: the two networks that predate v2 sign v1
// until the upgrade is applied on them, and v2 from then on; any other chain
// was born verifying v2.
func TestSignBytesVersionFor(t *testing.T) {
	cases := []struct {
		chain   string
		applied int64
		want    int
	}{
		{"qorechain-vladi", 0, 1},
		{"qorechain-diana", 0, 1},
		{"qorechain-vladi", 3_000_000, 2},
		{"qorechain-diana", 5_746_000, 2},
		{"qorechain-devnet", 0, 2},
		{"updev98", 0, 2},
	}
	for _, c := range cases {
		if got := SignBytesVersionFor(c.chain, c.applied); got != c.want {
			t.Errorf("SignBytesVersionFor(%q, %d) = %d, want %d", c.chain, c.applied, got, c.want)
		}
	}
}
