package wasmfilter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtestdata "github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// shapeModule is a valid module with the given numbers of imports, exports,
// globals, element segments (entries each) and data segments, and one
// function of type 0.
type shapeSpec struct {
	imports, exports, globals, elemSegs, elemEntries, dataSegs int
	types                                                      int // declared beyond the module's own 2
}

func shapeModule(s shapeSpec) []byte {
	m := []byte("\x00asm\x01\x00\x00\x00")
	types := [][]byte{t0, fn([]byte{i32}, nil)}
	for i := 0; i < s.types; i++ {
		types = append(types, fn([]byte{i64}, nil))
	}
	m = append(m, section(1, vec(types...))...)
	if s.imports > 0 {
		imps := make([][]byte, s.imports)
		for i := range imps {
			imp := append(ccName("env"), ccName("abort")...)
			imps[i] = append(imp, 0x00, 0x01) // func of type 1
		}
		m = append(m, section(2, vec(imps...))...)
	}
	m = append(m, section(3, vec(uleb(0)))...)
	if s.elemSegs > 0 {
		m = append(m, section(4, vec([]byte{0x70, 0x01, 0xC4, 0x13, 0xC4, 0x13}))...) // 2,500 entries
	}
	m = append(m, section(5, vec([]byte{0x00, 0x01}))...)
	globals := make([][]byte, s.globals)
	for i := range globals {
		globals[i] = []byte{i32, 0x01, 0x41, 0x00, 0x0B}
	}
	m = append(m, section(6, vec(globals...))...)
	exports := make([][]byte, s.exports)
	for i := range exports {
		exports[i] = append(append(ccName(fmt.Sprintf("e%d", i)), 0x00), uleb(uint64(s.imports))...)
	}
	m = append(m, section(7, vec(exports...))...)
	if s.elemSegs > 0 {
		segs := make([][]byte, s.elemSegs)
		for i := range segs {
			seg := append([]byte{0x00, 0x41, 0x00, 0x0B}, uleb(uint64(s.elemEntries))...)
			for k := 0; k < s.elemEntries; k++ {
				seg = append(seg, uleb(uint64(s.imports))...)
			}
			segs[i] = seg
		}
		m = append(m, section(9, vec(segs...))...)
	}
	m = append(m, section(10, vec(body(0x0B)))...)
	data := make([][]byte, s.dataSegs)
	for i := range data {
		data[i] = []byte{0x00, 0x41, 0x00, 0x0B, 0x01, 0x2A}
	}
	return append(m, section(11, vec(data...))...)
}

// Inspect counts the parts of a module the VM sets up on every call.
func TestInspectCountsModuleShape(t *testing.T) {
	st, err := Inspect(shapeModule(shapeSpec{imports: 3, exports: 5, globals: 7, elemSegs: 2, elemEntries: 11, dataSegs: 13}))
	if err != nil {
		t.Fatal(err)
	}
	if st.Imports != 3 || st.Exports != 5 || st.Globals != 7 || st.ElemSegments != 2 || st.ElemEntries != 22 || st.DataSegments != 13 || st.Tables != 1 || st.Types != 2 || st.TypeParams != 1 {
		t.Fatalf("stats %+v", st)
	}
}

// At the bounds a module passes; one more of any part is refused.
func TestCheckShapeBounds(t *testing.T) {
	at := shapeSpec{100, MaxModuleExports, MaxModuleGlobals, MaxModuleElemSegments, MaxModuleElemEntries / MaxModuleElemSegments, MaxModuleDataSegments, MaxModuleTypes - 2}
	st, err := Inspect(shapeModule(at))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckShape(st); err != nil {
		t.Fatalf("at the bounds: %v", err)
	}
	over := map[string]shapeSpec{}
	for name, f := range map[string]func(*shapeSpec){
		"function types":   func(s *shapeSpec) { s.types++ },
		"exports":          func(s *shapeSpec) { s.exports++ },
		"globals":          func(s *shapeSpec) { s.globals++ },
		"element segments": func(s *shapeSpec) { s.elemSegs++ },
		"element entries":  func(s *shapeSpec) { s.elemSegs, s.elemEntries = 1, MaxModuleElemEntries+1 },
		"data segments":    func(s *shapeSpec) { s.dataSegs++ },
	} {
		s := at
		f(&s)
		over[name] = s
	}
	for name, s := range over {
		st, err := Inspect(shapeModule(s))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := CheckShape(st); !errors.Is(err, ErrModuleShape) {
			t.Fatalf("one over in %s: %v", name, err)
		}
	}
}

// Every release and test contract is well inside the bounds.
func TestRealContractsWithinShapeBounds(t *testing.T) {
	codes := map[string][]byte{
		"reflect_2_0": wasmtestdata.ReflectContractWasm(),
		"reflect_1_1": wasmtestdata.MigrateReflectContractWasm(),
		"hackatom":    wasmtestdata.HackatomContractWasm(),
		"cyberpunk":   wasmtestdata.CyberpunkContractWasm(),
		"ibc_reflect": wasmtestdata.IBCReflectContractWasm(),
		"burner":      wasmtestdata.BurnerContractWasm(),
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
		if err := CheckShape(st); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		t.Logf("%-28s types %2d imports %2d exports %2d globals %d element %d/%3d data %3d", name, st.Types, st.Imports, st.Exports, st.Globals, st.ElemSegments, st.ElemEntries, st.DataSegments)
	}
}

// The sections Inspect now reads fail closed when malformed or outside the
// feature set.
func TestCheckMalformedModuleSections(t *testing.T) {
	head := func(secs ...[]byte) []byte {
		m := []byte("\x00asm\x01\x00\x00\x00")
		for _, s := range secs {
			m = append(m, s...)
		}
		return m
	}
	cases := map[string][]byte{
		"export kind 4":          head(section(7, vec(append(ccName("x"), 0x04, 0x00)))),
		"export count too large": head(section(7, []byte{0x7F})),
		"import kind 4":          head(section(2, vec(append(append(ccName("env"), ccName("x")...), 0x04, 0x00)))),
		"global no end":          head(section(6, vec([]byte{i32, 0x00, 0x41, 0x00}))),
		"global v128":            head(section(6, vec([]byte{0x7B, 0x00, 0x41, 0x00, 0x0B}))),
		"global init op":         head(section(6, vec([]byte{i32, 0x00, 0x6A, 0x0B}))),
		"elem passive":           head(section(9, vec([]byte{0x01, 0x00, 0x00}))),
		"elem entries past end":  head(section(9, vec([]byte{0x00, 0x41, 0x00, 0x0B, 0x05, 0x00}))),
		"data passive":           head(section(11, vec([]byte{0x01, 0x01, 0x2A}))),
		"data past end":          head(section(11, vec([]byte{0x00, 0x41, 0x00, 0x0B, 0x05, 0x2A}))),
		"table externref":        head(section(4, vec([]byte{0x6F, 0x00, 0x01}))),
		"memory shared":          head(section(5, vec([]byte{0x03, 0x01, 0x01}))),
		"start trailing":         head(section(8, []byte{0x00, 0x00})),
		"trailing in exports":    head(section(7, append(vec(append(ccName("x"), 0x00, 0x00)), 0x00))),
	}
	for name, m := range cases {
		err := Check(m)
		if !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrUnsupportedOpcode) {
			t.Errorf("%s: %v, want a refusal", name, err)
		}
	}
}

// Operators run with more than StackRegisterDepth values live are counted
// deep; a call counts the values below its arguments.
func TestInspectCountsDeepStackOperators(t *testing.T) {
	chain := func(depth int, op ...byte) []byte {
		var b []byte
		for i := 0; i < depth; i++ {
			b = append(b, 0x41, 0x00) // i32.const 0
		}
		for i := 0; i < 10; i++ {
			b = append(b, op...)
		}
		for i := 0; i < depth; i++ {
			b = append(b, 0x1A)
		}
		return append(b, 0x0B)
	}
	types := [][]byte{t0, fn([]byte{i32, i32, i32, i32, i32}, nil)}
	deep := func(code []byte) Stats {
		t.Helper()
		st, err := Inspect(mkModule(types, body(code...)))
		if err != nil {
			t.Fatal(err)
		}
		return st
	}
	if st := deep(chain(3, 0x45)); st.DeepOps != 0 {
		t.Fatalf("eqz at 3 live values: %d deep", st.DeepOps)
	}
	if st := deep(chain(4, 0x45)); st.DeepByClass[OpBitCount] != 10 {
		t.Fatalf("eqz at 4 live values: %+v", st.DeepByClass)
	}
	// A block keeps the values below it live; its end restores the height.
	blk := append([]byte{0x41, 0x00, 0x41, 0x00, 0x41, 0x00, 0x02, 0x40, 0x41, 0x00, 0x45, 0x1A, 0x0B, 0x1A, 0x1A, 0x1A}, 0x0B)
	if st := deep(blk); st.DeepByClass[OpBitCount] != 1 {
		t.Fatalf("eqz inside a block above three values: %+v", st.DeepByClass)
	}
	// A call of five arguments with nothing below them is not deep; with
	// four values below it is.
	args := []byte{0x41, 0x00, 0x41, 0x00, 0x41, 0x00, 0x41, 0x00, 0x41, 0x00, 0x10, 0x01}
	st, err := Inspect(mkModuleTyped(types, []uint32{0, 1}, body(append(append([]byte{}, args...), 0x0B)...), body(0x0B)))
	if err != nil {
		t.Fatal(err)
	}
	if st.DeepByClass[OpCall] != 0 {
		t.Fatalf("a call with nothing below its arguments: %+v", st.DeepByClass)
	}
	below := append(append([]byte{0x41, 0x00, 0x41, 0x00, 0x41, 0x00, 0x41, 0x00}, args...), 0x1A, 0x1A, 0x1A, 0x1A, 0x0B)
	if st, err = Inspect(mkModuleTyped(types, []uint32{0, 1}, body(below...), body(0x0B))); err != nil {
		t.Fatal(err)
	}
	if st.DeepByClass[OpCall] != 1 {
		t.Fatalf("a call with four values below: %+v", st.DeepByClass)
	}
	// After br the rest of the block is unreachable: the height drops to
	// the block's base.
	br := []byte{0x02, 0x40, 0x41, 0x00, 0x41, 0x00, 0x41, 0x00, 0x41, 0x00, 0x0C, 0x00, 0x41, 0x00, 0x45, 0x1A, 0x0B, 0x0B}
	if st := deep(br); st.DeepByClass[OpBitCount] != 0 {
		t.Fatalf("eqz after br: %+v", st.DeepByClass)
	}
}

// mkModuleTyped is mkModule with a type index per function.
func mkModuleTyped(types [][]byte, fnTypes []uint32, bodies ...[]byte) []byte {
	m := []byte("\x00asm\x01\x00\x00\x00")
	m = append(m, section(1, vec(types...))...)
	funcs := make([][]byte, len(fnTypes))
	for i, ti := range fnTypes {
		funcs[i] = uleb(uint64(ti))
	}
	m = append(m, section(3, vec(funcs...))...)
	return append(m, section(10, vec(bodies...))...)
}

// Stored code over the shape bounds is refused at every entry point while
// the filter and the shape switch are both on, and runs when either is off;
// without a context it is refused, as other failing code is. A filter
// without the switch wired never refuses for it.
func TestEngineRefusesShapeOnlyWhenOn(t *testing.T) {
	ctx := testCtx(t)
	on, shape := true, true
	vm := newFakeVM()
	e := New(func(sdk.Context) bool { return on }).WithShapeBounds(func(sdk.Context) bool { return shape }).DecorateEngine(vm).(*Engine)
	wide := vm.put(shapeModule(shapeSpec{exports: MaxModuleExports + 1}))
	narrow := vm.put(shapeModule(shapeSpec{exports: MaxModuleExports}))
	q := wasmkeeper.NewQueryHandler(ctx, nil, nil, nil)
	refused, errs := call(e, wide, q)
	if refused != entryPoints || !errors.Is(errs[0], ErrCodeRefused) {
		t.Fatalf("both on: %d of %d refused (%v)", refused, entryPoints, errs)
	}
	if refused, _ := call(e, narrow, q); refused != 0 {
		t.Fatalf("at the bound: %d refused", refused)
	}
	shape = false
	if refused, _ := call(e, wide, q); refused != 0 {
		t.Fatalf("shape switch off: %d refused", refused)
	}
	shape, on = true, false
	if refused, _ := call(e, wide, q); refused != 0 {
		t.Fatalf("filter off: %d refused", refused)
	}
	if refused, _ := call(e, wide, nil); refused != entryPoints {
		t.Fatalf("no context: %d refused", refused)
	}
	plain := New(func(sdk.Context) bool { return true }).DecorateEngine(newFakeVM()).(*Engine)
	if refused, _ := call(plain, plain.WasmEngine.(*fakeVM).put(shapeModule(shapeSpec{exports: MaxModuleExports + 1})), nil); refused != 0 {
		t.Fatalf("switch not wired: %d refused", refused)
	}
}

// Uploads over the shape bounds are refused while both switches are on.
func TestMsgServerRefusesShapeUploads(t *testing.T) {
	ctx := testCtx(t)
	shape := true
	inner := &fakeMsgServer{}
	s := New(func(sdk.Context) bool { return true }).WithShapeBounds(func(sdk.Context) bool { return shape }).DecorateMsgServer(inner)
	wide := shapeModule(shapeSpec{dataSegs: MaxModuleDataSegments + 1})
	if _, err := s.StoreCode(ctx, &wasmtypes.MsgStoreCode{WASMByteCode: gz(t, wide)}); !errors.Is(err, ErrCodeRefused) || inner.reached != 0 {
		t.Fatalf("over the bound: %v, %d reached", err, inner.reached)
	}
	if _, err := s.StoreCode(ctx, &wasmtypes.MsgStoreCode{WASMByteCode: shapeModule(shapeSpec{dataSegs: MaxModuleDataSegments})}); err != nil {
		t.Fatalf("at the bound: %v", err)
	}
	shape = false
	if _, err := s.StoreCode(ctx, &wasmtypes.MsgStoreCode{WASMByteCode: wide}); err != nil || inner.reached != 2 {
		t.Fatalf("switch off: %v, %d reached", err, inner.reached)
	}
}
