package wasmfilter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

// nopsModule is a valid module with one function of n nops.
func nopsModule(n int) []byte {
	m := []byte("\x00asm\x01\x00\x00\x00")
	m = append(m, section(1, []byte{1, 0x60, 0, 0})...)
	m = append(m, section(3, []byte{1, 0})...)
	body := append([]byte{0}, bytes.Repeat([]byte{0x01}, n)...)
	body = append(body, 0x0b)
	return append(m, section(10, append([]byte{1}, append(uleb(uint64(len(body))), body...)...))...)
}

func withSizeField(b []byte, declared uint32) []byte {
	tail := make([]byte, 4)
	binary.LittleEndian.PutUint32(tail, declared)
	return append(append([]byte{}, b...), tail...)
}

// Under upload bounds an upload unpacks only to its declared size, in one
// gzip member with nothing after it; honest gzip and raw uploads of any size
// are unaffected.
func TestInspectUploadBounded(t *testing.T) {
	big := nopsModule(800_000)
	tiny := nopsModule(1)
	for _, c := range []struct {
		name string
		code []byte
		ok   bool
	}{
		{"honest gzip", gz(t, big), true},
		{"honest gzip of a tiny module (larger than the module)", gz(t, tiny), true},
		{"raw", big, true},
		{"size field lies low, after the member", withSizeField(gz(t, big), 16), false},
		{"size field lies high, after the member", withSizeField(gz(t, tiny), 800_000), false},
		{"a second member", append(gz(t, tiny), gz(t, tiny)...), false},
		{"truncated", gz(t, big)[:len(gz(t, big))-4], false},
		{"broken data", append([]byte{0x1f, 0x8b, 8, 0, 0, 0, 0, 0, 0, 0xff}, bytes.Repeat([]byte{0xff}, 40)...), false},
	} {
		declared, _ := DeclaredUploadSize(c.code)
		insp, err := InspectUploadBounded(c.code, declared)
		if c.ok {
			if err != nil || insp.Err != nil || uint64(len(insp.Raw)) != declared {
				t.Errorf("%s: %v %v (%d of %d bytes)", c.name, err, insp.Err, len(insp.Raw), declared)
			}
			continue
		}
		if !errors.Is(err, ErrUploadSize) {
			t.Errorf("%s: %v, want refused", c.name, err)
		}
	}
	// A size field after the member that names the real size still makes
	// the upload refused (the field read must be the member's own).
	if _, err := InspectUploadBounded(withSizeField(gz(t, big), uint32(len(big))), uint64(len(big))); !errors.Is(err, ErrUploadSize) {
		t.Errorf("trailing bytes naming the real size: %v", err)
	}
}

// The work a lying upload causes is bounded by what it declared (and paid
// for): an 800 KiB module declared as 16 bytes stops after 17.
func TestInspectUploadBoundedStopsAtTheDeclaredSize(t *testing.T) {
	lie := withSizeField(gz(t, nopsModule(800_000)), 16)
	honest := gz(t, nopsModule(800_000))
	const n = 20
	t0 := time.Now()
	for i := 0; i < n; i++ {
		if _, err := InspectUploadBounded(lie, 16); err == nil {
			t.Fatal("accepted")
		}
	}
	lieT := time.Since(t0) / n
	t0 = time.Now()
	for i := 0; i < n; i++ {
		if _, err := InspectUploadBounded(honest, 800_000+30); err != nil && !errors.Is(err, ErrUploadSize) {
			t.Fatal(err)
		}
	}
	honestT := time.Since(t0) / n
	t.Logf("refusing the lie: %v; unpacking and inspecting the honest 800 KB module: %v", lieT, honestT)
	if lieT*20 > honestT {
		t.Fatalf("the lie cost %v against %v for the whole module", lieT, honestT)
	}
}
