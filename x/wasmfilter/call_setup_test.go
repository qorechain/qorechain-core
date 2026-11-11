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

// The high-water mark is the initial pages without memory.grow, else the
// VM's limit or a lower declared maximum; the setup units follow it and
// the data segments.
func TestCallSetupUnits(t *testing.T) {
	for _, c := range []struct {
		pages, max uint32
		grow       bool
		segs       int
		hw         uint64
	}{
		{4, 0, false, 0, 4},
		{64, 0, false, 1020, 64},
		{17, 0, true, 0, VMMemoryLimitPages},
		{17, 100, true, 2, 100},
		{17, 10, true, 0, 17}, // a maximum below the minimum: the minimum
		{17, 4000, true, 0, VMMemoryLimitPages},
	} {
		st, err := Inspect(memModule(c.pages, c.max, c.grow, c.segs, 1))
		if err != nil && c.max >= c.pages {
			t.Fatal(err)
		}
		if got := MemoryHighWaterPages(st); got != c.hw {
			t.Fatalf("%+v: high-water %d, want %d", c, got, c.hw)
		}
		charged := min(c.hw, uint64(c.pages)+CallSetupGrowPages)
		if got := ChargedSetupPages(st); got != charged {
			t.Fatalf("%+v: charged %d pages, want %d", c, got, charged)
		}
		want := CallSetupUnitsBase + charged*CallSetupUnitsPerPage + uint64(c.segs)*CallSetupUnitsPerDataSegment
		if got := CallSetupUnits(st); got != want {
			t.Fatalf("%+v: units %d, want %d", c, got, want)
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
		t.Logf("%-28s pages %2d max %v/%d grow %d data %3d (%6d B): high-water %3d, charged %2d, setup %5d units",
			name, st.MemoryPages, st.MemoryHasMax, st.MemoryMaxPages, st.GrowOps, st.DataSegments, st.DataBytes, MemoryHighWaterPages(st), ChargedSetupPages(st), CallSetupUnits(st))
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
// call, 1.93 us per 4 KiB page touched for the first time. Every shape whose
// touched pages are the ones it declares (initial memory, data segments) is
// charged at least its time at the 265 ns design bound; a module that can
// grow is charged CallSetupGrowPages more.
func TestCallSetupUnitsCoverDeclaredMemory(t *testing.T) {
	const callNs, pageNs, bound = 83_000.0, 1_930.0, 265.0
	for _, c := range []struct {
		name      string
		pages     uint32
		segs      int
		touchedOS int
	}{
		{"minimal", 4, 0, 0},
		{"64 pages, all touched", MaxModuleInitialPages, 0, MaxModuleInitialPages * 16},
		{"64 pages, 1,020 one-page segments", MaxModuleInitialPages, 1020, 1020},
		{"64 pages, segments and all touched", MaxModuleInitialPages, 1020, MaxModuleInitialPages * 16},
		{"17 pages, all touched", 17, 3, 17 * 16},
	} {
		st, err := Inspect(memModule(c.pages, 0, false, c.segs, 1))
		if err != nil {
			t.Fatal(err)
		}
		ns := callNs + pageNs*float64(c.touchedOS)
		per := ns / float64(CallSetupUnits(st))
		t.Logf("%-36s %6d units, modelled %7.0f us on AMD: %5.0f ns/unit", c.name, CallSetupUnits(st), ns/1e3, per)
		if per > bound {
			t.Errorf("%s: %.0f ns per unit", c.name, per)
		}
	}
}
