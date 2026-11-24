package wasmfilter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"runtime"
	"testing"
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
//
// Work is measured as the bytes the check allocates (what unpacking and
// inspecting cost in memory, and what it touches), not as a ratio of wall
// clock times, which a loaded machine distorts (confirmation review 2, I5).
// Allocation is deterministic for a given input, so the bounds below are
// exact properties, not timing luck:
//   - refusing the lie costs the same whatever the real size behind it
//     (an 800 KB module and one four times larger), so the work does not
//     grow with what the upload would unpack to;
//   - it costs a small fraction of unpacking and inspecting the honest
//     800 KB module.
func TestInspectUploadBoundedStopsAtTheDeclaredSize(t *testing.T) {
	lie := withSizeField(gz(t, nopsModule(800_000)), 16)
	lieLarger := withSizeField(gz(t, nopsModule(3_200_000)), 16)
	honest := gz(t, nopsModule(800_000))
	const n = 20
	lieB := allocPerRun(n, func() {
		if _, err := InspectUploadBounded(lie, 16); err == nil {
			t.Fatal("accepted")
		}
	})
	lieLargerB := allocPerRun(n, func() {
		if _, err := InspectUploadBounded(lieLarger, 16); err == nil {
			t.Fatal("accepted")
		}
	})
	honestB := allocPerRun(n, func() {
		if _, err := InspectUploadBounded(honest, 800_000+30); err != nil && !errors.Is(err, ErrUploadSize) {
			t.Fatal(err)
		}
	})
	t.Logf("bytes allocated per check: lie over 800 KB %d, lie over 3.2 MB %d, honest 800 KB %d", lieB, lieLargerB, honestB)
	if lieLargerB > lieB+lieB/4 {
		t.Fatalf("the lie's work grows with the real size: %d bytes over 800 KB, %d over 3.2 MB", lieB, lieLargerB)
	}
	if lieB*20 > honestB {
		t.Fatalf("the lie cost %d bytes against %d for the whole module", lieB, honestB)
	}
}

// allocPerRun is the average number of bytes f allocates per call over n
// calls (after one warm-up call).
func allocPerRun(n int, f func()) uint64 {
	f()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for i := 0; i < n; i++ {
		f()
	}
	runtime.ReadMemStats(&after)
	return (after.TotalAlloc - before.TotalAlloc) / uint64(n)
}
