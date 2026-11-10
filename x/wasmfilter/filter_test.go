package wasmfilter

import (
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	wasmtestdata "github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
)

// ---- a minimal module builder ----------------------------------------------

func uleb(v uint64) []byte {
	var out []byte
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			out = append(out, b|0x80)
			continue
		}
		return append(out, b)
	}
}

func vec(items ...[]byte) []byte {
	out := uleb(uint64(len(items)))
	for _, it := range items {
		out = append(out, it...)
	}
	return out
}

func section(id byte, body []byte) []byte {
	return append(append([]byte{id}, uleb(uint64(len(body)))...), body...)
}

// fn is a function type: params and results as value type bytes.
func fn(params, results []byte) []byte {
	out := []byte{0x60}
	out = append(out, uleb(uint64(len(params)))...)
	out = append(out, params...)
	out = append(out, uleb(uint64(len(results)))...)
	return append(out, results...)
}

// body is a function body with no locals; code must end with the function's
// own 0x0B.
func body(code ...byte) []byte {
	b := append([]byte{0x00}, code...)
	return append(uleb(uint64(len(b))), b...)
}

// mkModule builds header + type section + function section + code section.
// Function i has type 0.
func mkModule(types [][]byte, bodies ...[]byte) []byte {
	m := []byte("\x00asm\x01\x00\x00\x00")
	m = append(m, section(1, vec(types...))...)
	funcs := make([][]byte, len(bodies))
	for i := range funcs {
		funcs[i] = uleb(0)
	}
	m = append(m, section(3, vec(funcs...))...)
	return append(m, section(10, vec(bodies...))...)
}

const (
	i32 = 0x7F
	i64 = 0x7E
)

var (
	t0     = fn(nil, nil)                   // [] -> []
	tParam = fn([]byte{i32}, []byte{i32})   // [i32] -> [i32]
	tMulti = fn(nil, []byte{i32, i32})      // [] -> [i32 i32]
	tOne   = fn(nil, []byte{i32})           // [] -> [i32], same as the 0x7F short form
	tTwoIn = fn([]byte{i32, i64}, []byte{}) // [i32 i64] -> []
)

func wantReason(t *testing.T, name string, err, want error) {
	t.Helper()
	if want == nil {
		if err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
		return
	}
	if !errors.Is(err, want) {
		t.Errorf("%s: got %v, want %v", name, err, want)
	}
}

// ---- the refused constructs and their accepted equivalents ------------------

func TestCheckBlockTypes(t *testing.T) {
	types := [][]byte{t0, tParam, tMulti, tOne, tTwoIn}
	cases := []struct {
		name string
		code []byte
		want error
	}{
		// Accepted: the forms compilers emit for contracts.
		{"empty", []byte{0x0B}, nil},
		{"block empty", []byte{0x02, 0x40, 0x0B, 0x0B}, nil},
		{"block i32", []byte{0x02, i32, 0x41, 0x01, 0x0B, 0x1A, 0x0B}, nil},
		{"loop empty", []byte{0x03, 0x40, 0x0B, 0x0B}, nil},
		{"if empty no else", []byte{0x41, 0x00, 0x04, 0x40, 0x0B, 0x0B}, nil},
		{"if i32 with else", []byte{0x41, 0x00, 0x04, i32, 0x41, 0x01, 0x05, 0x41, 0x02, 0x0B, 0x1A, 0x0B}, nil},
		{"nested", []byte{0x02, 0x40, 0x03, 0x40, 0x41, 0x00, 0x04, 0x40, 0x0C, 0x02, 0x0B, 0x0B, 0x0B, 0x0B}, nil},

		// The narrow case: if, type-index signature with parameters, no else.
		{"if params no else", []byte{0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x0B, 0x1A, 0x0B}, ErrParamIfWithoutElse},
		{"if two params no else", []byte{0x41, 0x01, 0x42, 0x01, 0x41, 0x00, 0x04, 0x04, 0x0B, 0x0B}, ErrParamIfWithoutElse},
		{"nested if params no else", []byte{0x02, 0x40, 0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x0B, 0x1A, 0x0B, 0x0B}, ErrParamIfWithoutElse},
		// Every other type-index block type.
		{"if params with else", []byte{0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x05, 0x0B, 0x1A, 0x0B}, ErrTypedBlock},
		{"block params", []byte{0x41, 0x01, 0x02, 0x01, 0x0B, 0x1A, 0x0B}, ErrTypedBlock},
		{"loop params", []byte{0x41, 0x01, 0x03, 0x01, 0x0B, 0x1A, 0x0B}, ErrTypedBlock},
		{"block multi result", []byte{0x02, 0x02, 0x41, 0x01, 0x41, 0x02, 0x0B, 0x1A, 0x1A, 0x0B}, ErrTypedBlock},
		{"if multi result", []byte{0x41, 0x00, 0x04, 0x02, 0x41, 0x01, 0x41, 0x02, 0x05, 0x41, 0x03, 0x41, 0x04, 0x0B, 0x1A, 0x1A, 0x0B}, ErrTypedBlock},
		// Even a type index that means exactly what 0x40 or 0x7F means.
		{"block type index []->[]", []byte{0x02, 0x00, 0x0B, 0x0B}, ErrTypedBlock},
		{"block type index []->[i32]", []byte{0x02, 0x03, 0x41, 0x01, 0x0B, 0x1A, 0x0B}, ErrTypedBlock},
		// A type index in multi-byte LEB form is still a type index.
		{"block type index padded", []byte{0x02, 0x80, 0x00, 0x0B, 0x0B}, ErrTypedBlock},
		{"block type index out of range", []byte{0x02, 0x09, 0x0B, 0x0B}, ErrMalformed},
		{"block type v128", []byte{0x02, 0x7B, 0x0B, 0x0B}, ErrUnsupportedOpcode},
		{"block type funcref", []byte{0x02, 0x70, 0x0B, 0x0B}, ErrUnsupportedOpcode},
	}
	for _, c := range cases {
		wantReason(t, c.name, Check(mkModule(types, body(c.code...))), c.want)
	}
}

func TestCheckParamIfReportedEvenAfterOtherTypedBlocks(t *testing.T) {
	// A body with a typed block first and a param if without else later is
	// reported with the narrower reason.
	code := []byte{0x02, 0x02, 0x41, 0x01, 0x41, 0x02, 0x0B, 0x1A, 0x1A,
		0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x0B, 0x1A, 0x0B}
	wantReason(t, "typed then param if", Check(mkModule([][]byte{t0, tParam, tMulti}, body(code...))), ErrParamIfWithoutElse)
	// And in a later function than the first refused one.
	m := mkModule([][]byte{t0, tParam}, body(0x0B), body(0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x0B, 0x1A, 0x0B))
	if err := Check(m); !errors.Is(err, ErrParamIfWithoutElse) || !strings.Contains(err.Error(), "function body 1") {
		t.Fatalf("got %v", err)
	}
}

func brTable(n int) []byte {
	code := []byte{0x02, 0x40, 0x41, 0x00, 0x0E}
	code = append(code, uleb(uint64(n))...)
	for i := 0; i <= n; i++ {
		code = append(code, 0x00)
	}
	return append(code, 0x0B, 0x0B)
}

func TestCheckBrTableCap(t *testing.T) {
	types := [][]byte{t0}
	for _, n := range []int{0, 1, 84, MaxBrTableTargets} {
		st, err := Inspect(mkModule(types, body(brTable(n)...)))
		if err != nil || st.MaxBrTable != n {
			t.Errorf("br_table %d: %v (max %d)", n, err, st.MaxBrTable)
		}
	}
	for _, n := range []int{MaxBrTableTargets + 1, 100_000} {
		wantReason(t, "br_table over cap", Check(mkModule(types, body(brTable(n)...))), ErrBrTableTooLarge)
	}
	// A declared count far above the cap is refused before its targets are
	// read, so a short body cannot claim billions of them.
	code := append([]byte{0x02, 0x40, 0x41, 0x00, 0x0E}, uleb(0xFFFFFFFF)...)
	wantReason(t, "br_table huge count", Check(mkModule(types, body(code...))), ErrBrTableTooLarge)
}

func TestCheckOpcodesOutsideTheFeatureSet(t *testing.T) {
	types := [][]byte{t0}
	cases := map[string][]byte{
		"simd prefix":            {0xFD, 0x0C},
		"threads prefix":         {0xFE, 0x00},
		"memory.copy":            {0xFC, 0x0A, 0x00, 0x00},
		"memory.fill":            {0xFC, 0x0B, 0x00},
		"memory.init":            {0xFC, 0x08, 0x00, 0x00},
		"table.grow":             {0xFC, 0x0F, 0x00},
		"typed select":           {0x1C, 0x01, i32},
		"ref.null":               {0xD0, 0x70},
		"ref.func":               {0xD2, 0x00},
		"table.get":              {0x25, 0x00},
		"return_call":            {0x12, 0x00},
		"try":                    {0x06, 0x40},
		"throw":                  {0x08, 0x00},
		"call_indirect table 1":  {0x11, 0x00, 0x01},
		"memory.size memory 1":   {0x3F, 0x01},
		"load with memory index": {0x41, 0x00, 0x28, 0x42, 0x00, 0x00},
		"unassigned 0x27":        {0x27},
		"unassigned 0xC5":        {0xC5},
	}
	for name, ops := range cases {
		code := append(append([]byte{}, ops...), 0x0B)
		wantReason(t, name, Check(mkModule(types, body(code...))), ErrUnsupportedOpcode)
	}
	// Accepted: sign extension and the saturating conversions.
	for _, ops := range [][]byte{
		{0x41, 0x00, 0xC0, 0x1A},
		{0x42, 0x00, 0xC4, 0x1A},
		{0x43, 0, 0, 0, 0, 0xFC, 0x00, 0x1A},
		{0x44, 0, 0, 0, 0, 0, 0, 0, 0, 0xFC, 0x07, 0x1A},
		{0x41, 0x00, 0x28, 0x02, 0x00, 0x1A},
		{0x3F, 0x00, 0x1A},
		{0x41, 0x00, 0x40, 0x00, 0x1A},
	} {
		code := append(append([]byte{}, ops...), 0x0B)
		wantReason(t, "accepted op", Check(mkModule(types, body(code...))), nil)
	}
}

func TestCheckMalformed(t *testing.T) {
	good := mkModule([][]byte{t0}, body(0x41, 0x2A, 0x1A, 0x0B))
	if err := Check(good); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"empty":                   nil,
		"no header":               []byte("\x00asm"),
		"bad magic":               []byte("\x00wasm\x01\x00\x00"),
		"version 2":               []byte("\x00asm\x02\x00\x00\x00"),
		"component":               []byte("\x00asm\x0d\x00\x01\x00"),
		"unknown section 13":      append([]byte("\x00asm\x01\x00\x00\x00"), section(13, nil)...),
		"section past end":        append([]byte("\x00asm\x01\x00\x00\x00"), 0x01, 0x10, 0x00),
		"section size 6-byte LEB": append([]byte("\x00asm\x01\x00\x00\x00"), 0x00, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00),
		"duplicate type section":  append(append([]byte("\x00asm\x01\x00\x00\x00"), section(1, vec(t0))...), section(1, vec(t0))...),
		"type not a func":         mkModule([][]byte{{0x5F, 0x00}}, body(0x0B)),
		"type v128 param":         mkModule([][]byte{fn([]byte{0x7B}, nil)}, body(0x0B)),
		"type externref":          mkModule([][]byte{fn(nil, []byte{0x6F})}, body(0x0B)),
		"else without if":         mkModule([][]byte{t0}, body(0x05, 0x0B)),
		"else in block":           mkModule([][]byte{t0}, body(0x02, 0x40, 0x05, 0x0B, 0x0B)),
		"two elses":               mkModule([][]byte{t0}, body(0x41, 0x00, 0x04, 0x40, 0x05, 0x05, 0x0B, 0x0B)),
		"missing end":             mkModule([][]byte{t0}, body(0x02, 0x40, 0x0B)),
		"bytes after end":         mkModule([][]byte{t0}, body(0x0B, 0x01)),
		"extra end":               mkModule([][]byte{t0}, body(0x0B, 0x0B)),
		"i32.const too wide":      mkModule([][]byte{t0}, body(0x41, 0xFF, 0xFF, 0xFF, 0xFF, 0x0F, 0x1A, 0x0B)),
		"i64.const 11 bytes":      mkModule([][]byte{t0}, body(0x42, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00, 0x1A, 0x0B)),
		"f64.const short":         mkModule([][]byte{t0}, body(0x44, 0, 0, 0)),
		"local type v128":         append(append(append([]byte("\x00asm\x01\x00\x00\x00"), section(1, vec(t0))...), section(3, vec(uleb(0)))...), section(10, vec([]byte{0x04, 0x01, 0x01, 0x7B, 0x0B}))...),
		"custom name past end":    append([]byte("\x00asm\x01\x00\x00\x00"), section(0, []byte{0x05, 'a'})...),
		"code count too large":    append(append([]byte("\x00asm\x01\x00\x00\x00"), section(1, vec(t0))...), section(10, []byte{0x7F})...),
		"type count too large":    append([]byte("\x00asm\x01\x00\x00\x00"), section(1, []byte{0x7F, 0x60, 0x00, 0x00})...),
	}
	for name, m := range cases {
		err := Check(m)
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		if !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrUnsupportedOpcode) {
			t.Errorf("%s: %v, want a malformed refusal", name, err)
		}
	}
}

// Every truncation of a real contract fails closed, and none panics. A cut
// that lands exactly on a section boundary leaves a well-formed (smaller)
// module, so only cuts inside a section are required to fail.
func TestCheckEveryTruncationOfARealContract(t *testing.T) {
	code := wasmtestdata.HackatomContractWasm()
	if err := Check(code); err != nil {
		t.Fatal(err)
	}
	boundaries := map[int]bool{8: true}
	r := &reader{b: code, pos: 8}
	for !r.eof() {
		_, _ = r.byte()
		n, _ := r.u32()
		r.pos += int(n)
		boundaries[r.pos] = true
	}
	// Every cut in the header and the first sections, every cut next to a
	// section boundary, and a spread over the rest (all cuts would be
	// quadratic in the contract size).
	cuts := map[int]bool{}
	for c := 0; c < 4096; c++ {
		cuts[c] = true
	}
	for b := range boundaries {
		for d := -3; d <= 3; d++ {
			cuts[b+d] = true
		}
	}
	for c := 4096; c < len(code); c += 37 {
		cuts[c] = true
	}
	for cut := range cuts {
		if cut < 0 || cut >= len(code) {
			continue
		}
		if err := Check(code[:cut]); err == nil && !boundaries[cut] {
			t.Fatalf("truncated at %d of %d: accepted", cut, len(code))
		}
	}
}

func FuzzCheck(f *testing.F) {
	f.Add(wasmtestdata.HackatomContractWasm()[:4096])
	f.Add(mkModule([][]byte{t0, tParam}, body(0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x0B, 0x1A, 0x0B)))
	f.Add(mkModule([][]byte{t0}, body(brTable(3)...)))
	f.Fuzz(func(t *testing.T, b []byte) {
		err1 := Check(b)
		err2 := Check(b)
		if (err1 == nil) != (err2 == nil) {
			t.Fatal("non-deterministic verdict")
		}
	})
}

// ---- gzip, as wasmd unpacks MsgStoreCode ------------------------------------

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCheckUploadGzip(t *testing.T) {
	bad := mkModule([][]byte{t0, tParam}, body(0x41, 0x01, 0x41, 0x00, 0x04, 0x01, 0x0B, 0x1A, 0x0B))
	good := wasmtestdata.ReflectContractWasm()

	if checked, err := CheckUpload(gz(t, bad)); !checked || !errors.Is(err, ErrParamIfWithoutElse) {
		t.Fatalf("gzip bad: %v %v", checked, err)
	}
	if checked, err := CheckUpload(bad); !checked || !errors.Is(err, ErrParamIfWithoutElse) {
		t.Fatalf("raw bad: %v %v", checked, err)
	}
	if checked, err := CheckUpload(gz(t, good)); !checked || err != nil {
		t.Fatalf("gzip good: %v %v", checked, err)
	}
	// Bytes that start like gzip but do not unpack are wasmd's to refuse.
	broken := gz(t, good)
	broken = broken[:len(broken)/2]
	if checked, err := CheckUpload(broken); checked || err != nil {
		t.Fatalf("broken gzip: %v %v", checked, err)
	}
}

// ---- real contracts ---------------------------------------------------------

// reflect_2_0 in wasmd's test data is byte-identical to the reflect build
// stored on diana (sha256 99aef385...), and hackatom to the devnet one.
func TestRealContractsPass(t *testing.T) {
	for name, code := range map[string][]byte{
		"reflect_2_0 (diana reflect)": wasmtestdata.ReflectContractWasm(),
		"reflect_1_1":                 wasmtestdata.MigrateReflectContractWasm(),
		"hackatom":                    wasmtestdata.HackatomContractWasm(),
		"cyberpunk":                   wasmtestdata.CyberpunkContractWasm(),
		"ibc_reflect":                 wasmtestdata.IBCReflectContractWasm(),
		"burner":                      wasmtestdata.BurnerContractWasm(),
	} {
		st, err := Inspect(code)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if st.TypedBlocks != 0 || st.MaxBrTable > 84 || st.Functions == 0 {
			t.Errorf("%s: unexpected stats %+v", name, st)
		}
	}
}

// TestCorpus checks every file in the directories listed in
// QORE_WASM_CORPUS (path-list separated). Files named *.fail.wasm must fail,
// everything else must pass. Skipped when unset.
func TestCorpus(t *testing.T) {
	dirs := os.Getenv("QORE_WASM_CORPUS")
	if dirs == "" {
		t.Skip("QORE_WASM_CORPUS not set")
	}
	n := 0
	for _, dir := range filepath.SplitList(dirs) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			name := e.Name()
			if !e.Type().IsRegular() || !(strings.HasSuffix(name, ".wasm") || strings.HasSuffix(name, ".wasm.gz")) {
				continue
			}
			bz, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			checked, verr := CheckUpload(bz)
			if !checked {
				t.Errorf("%s: cannot unpack", name)
				continue
			}
			n++
			if wantFail := strings.HasSuffix(name, ".fail.wasm"); wantFail != (verr != nil) {
				t.Errorf("%s: verdict %v", name, verr)
			}
		}
	}
	t.Logf("%d corpus files checked", n)
}
