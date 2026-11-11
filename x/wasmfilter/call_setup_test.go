package wasmfilter

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	wasmtestdata "github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
)

// memModule is a valid module with one memory of pages (and a maximum when
// maxPages > 0), one function that grows the memory when grow is set, and
// segs data segments of n bytes each.
func memModule(pages, maxPages uint32, grow bool, segs, n int) []byte {
	m := []byte("\x00asm\x01\x00\x00\x00")
	m = append(m, section(1, vec(t0))...)
	m = append(m, section(3, vec(uleb(0)))...)
	lim := append([]byte{0x00}, uleb(uint64(pages))...)
	if maxPages > 0 {
		lim = append(append([]byte{0x01}, uleb(uint64(pages))...), uleb(uint64(maxPages))...)
	}
	m = append(m, section(5, vec(lim))...)
	code := body(0x0B)
	if grow {
		code = body(0x41, 0x01, 0x40, 0x00, 0x1A, 0x0B) // i32.const 1; memory.grow; drop
	}
	m = append(m, section(10, vec(code))...)
	data := make([][]byte, segs)
	for i := range data {
		data[i] = append(append([]byte{0x00, 0x41, 0x00, 0x0B}, uleb(uint64(n))...), make([]byte, n)...)
	}
	return append(m, section(11, vec(data...))...)
}

// Inspect records the initial pages, the declared maximum, the memory.grow
// operators and the data bytes.
func TestInspectRecordsMemory(t *testing.T) {
	st, err := Inspect(memModule(20, 100, true, 3, 7))
	if err != nil {
		t.Fatal(err)
	}
	if st.MemoryPages != 20 || !st.MemoryHasMax || st.MemoryMaxPages != 100 || st.GrowOps != 1 || st.DataSegments != 3 || st.DataBytes != 21 {
		t.Fatalf("stats %+v", st)
	}
	if st, _ = Inspect(memModule(17, 0, false, 0, 0)); st.MemoryPages != 17 || st.MemoryHasMax || st.GrowOps != 0 {
		t.Fatalf("stats %+v", st)
	}
}

// At most MaxModuleInitialPages of initial memory.
func TestCheckShapeInitialPages(t *testing.T) {
	for pages, ok := range map[uint32]bool{17: true, MaxModuleInitialPages: true, MaxModuleInitialPages + 1: false, 512: false} {
		st, err := Inspect(memModule(pages, 0, false, 0, 0))
		if err != nil {
			t.Fatal(err)
		}
		err = CheckShape(st)
		if ok != (err == nil) || (!ok && (!errors.Is(err, ErrModuleShape) || !strings.Contains(err.Error(), "initial memory pages"))) {
			t.Fatalf("%d pages: %v", pages, err)
		}
	}
}

// The setup charge follows the initial memory and the data segments; the
// growth charge the pages a call grew by, up to the VM's limit.
func TestCallSetupUnits(t *testing.T) {
	for _, c := range []struct {
		pages uint32
		grow  bool
		segs  int
	}{{4, false, 0}, {64, false, 1020}, {17, true, 0}, {17, true, 22}} {
		st, err := Inspect(memModule(c.pages, 0, c.grow, c.segs, 1))
		if err != nil {
			t.Fatal(err)
		}
		want := CallSetupUnitsBase + uint64(c.pages)*CallSetupUnitsPerPage + uint64(c.segs)*CallSetupUnitsPerDataSegment
		if got := CallSetupUnits(st); got != want {
			t.Fatalf("%+v: units %d, want %d", c, got, want)
		}
	}
	for _, c := range []struct{ initial, reached, want uint64 }{
		{17, 0, 0},  // no report
		{17, 17, 0}, // did not grow
		{17, 18, CallSetupUnitsPerPage},
		{17, 512, 495 * CallSetupUnitsPerPage},
		{17, 4000, 495 * CallSetupUnitsPerPage}, // never past the VM's limit
	} {
		if got := CallGrowthUnits(c.initial, c.reached); got != c.want {
			t.Fatalf("%d -> %d pages: %d units, want %d", c.initial, c.reached, got, c.want)
		}
	}
}

// Every release and test contract declares at most 20 initial pages and no
// maximum.
func TestRealContractsCallSetup(t *testing.T) {
	codes := map[string][]byte{
		"reflect_2_0": wasmtestdata.ReflectContractWasm(),
		"hackatom":    wasmtestdata.HackatomContractWasm(),
		"cyberpunk":   wasmtestdata.CyberpunkContractWasm(),
		"ibc_reflect": wasmtestdata.IBCReflectContractWasm(),
	}
	rel, _ := filepath.Glob(os.Getenv("QORE_HANDLER_ROOTS") + "/app/testdata/*.wasm.gz")
	for _, f := range rel {
		codes[filepath.Base(f)] = ccRead(f)
	}
	for name, code := range codes {
		st, err := Inspect(code)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if st.MemoryPages > 20 || st.MemoryPages == 0 {
			t.Errorf("%s: %d initial pages", name, st.MemoryPages)
		}
		t.Logf("%-28s pages %2d max %v/%d grow %d data %3d (%6d B): setup %5d units",
			name, st.MemoryPages, st.MemoryHasMax, st.MemoryMaxPages, st.GrowOps, st.DataSegments, st.DataBytes, CallSetupUnits(st))
	}
}

// An upload takes its bytes below wasm-upload-compiled-size, and from it the
// larger of its bytes and its estimate over CompiledBytesPerUploadUnit;
// every upload within the caps takes at most MaxUploadUnits.
func TestUploadUnits(t *testing.T) {
	if got := uploadUnits(1000, 64_000, false); got != 1000 {
		t.Fatal(got)
	}
	if got := uploadUnits(1000, 64_000, true); got != 2000 {
		t.Fatal(got)
	}
	if got := uploadUnits(1000, 16_000, true); got != 1000 {
		t.Fatal(got)
	}
	if got := uploadUnits(800*1024, MaxUploadCompiledSize, true); got != MaxUploadUnits || MaxUploadUnits < 800*1024 {
		t.Fatalf("%d units at the caps, MaxUploadUnits %d", got, MaxUploadUnits)
	}
}

// The native AMD m6a cost model (the slower validator host): 83 us per
// call, 1.93 us per 4 KiB page touched for the first time. Every shape,
// whether it touches the memory it declares or memory it grew into, is
// charged at least its time at the 265 ns design bound.
func TestCallSetupUnitsCoverTouchedMemory(t *testing.T) {
	const callNs, pageNs, bound = 83_000.0, 1_930.0, 265.0
	for _, c := range []struct {
		name      string
		pages     uint32
		segs      int
		reached   uint64
		touchedOS int
	}{
		{"minimal", 4, 0, 4, 0},
		{"64 pages, all touched", MaxModuleInitialPages, 0, MaxModuleInitialPages, MaxModuleInitialPages * 16},
		{"64 pages, 1,020 one-page segments", MaxModuleInitialPages, 1020, MaxModuleInitialPages, 1020},
		{"64 pages, segments and all touched", MaxModuleInitialPages, 1020, MaxModuleInitialPages, MaxModuleInitialPages * 16},
		{"17 pages, all touched", 17, 3, 17, 17 * 16},
		{"17 pages, grow to 512, all touched", 17, 0, VMMemoryLimitPages, VMMemoryLimitPages * 16},
		{"64 pages + segments, grow to 512, all touched", MaxModuleInitialPages, 1020, VMMemoryLimitPages, VMMemoryLimitPages * 16},
		{"17 pages, grow 1, all touched", 17, 0, 18, 18 * 16},
	} {
		st, err := Inspect(memModule(c.pages, 0, c.reached > uint64(c.pages), c.segs, 1))
		if err != nil {
			t.Fatal(err)
		}
		units := CallSetupUnits(st) + CallGrowthUnits(uint64(c.pages), c.reached)
		ns := callNs + pageNs*float64(c.touchedOS)
		per := ns / float64(units)
		t.Logf("%-46s %6d units, modelled %7.0f us on AMD: %5.0f ns/unit", c.name, units, ns/1e3, per)
		if per > bound {
			t.Errorf("%s: %.0f ns per unit", c.name, per)
		}
	}
}
