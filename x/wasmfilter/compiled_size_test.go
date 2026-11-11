package wasmfilter

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wasmtestdata "github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	wasmvm "github.com/CosmWasm/wasmvm/v2"
)

func ccVec(items ...[]byte) []byte {
	out := uleb(uint64(len(items)))
	for _, it := range items {
		out = append(out, it...)
	}
	return out
}
func ccName(s string) []byte { return append(uleb(uint64(len(s))), s...) }
func ccRep(n int, p ...byte) []byte {
	var o []byte
	for i := 0; i < n; i++ {
		o = append(o, p...)
	}
	return o
}

func ccModule(n int, locals []byte, body []byte) []byte {
	typeSec := ccVec([]byte{0x60, 1, 0x7f, 1, 0x7f}, []byte{0x60, 1, 0x7f, 0}, []byte{0x60, 0, 0}, []byte{0x60, 3, 0x7f, 0x7f, 0x7f, 1, 0x7f})
	funcs := [][]byte{{0}, {1}, {2}, {3}}
	for i := 0; i < n; i++ {
		funcs = append(funcs, []byte{2})
	}
	tableSec := ccVec([]byte{0x70, 0x01, 0x01, 0x01})
	memSec := ccVec([]byte{0x00, 4})
	globSec := ccVec([]byte{0x7f, 0x01, 0x41, 0x80, 0x80, 0x04, 0x0b})
	exp := func(nm string, kind byte, idx uint64) []byte { return append(append(ccName(nm), kind), uleb(idx)...) }
	expSec := ccVec(exp("memory", 2, 0), exp("allocate", 0, 0), exp("deallocate", 0, 1), exp("interface_version_8", 0, 2), exp("instantiate", 0, 3))
	wrap := func(b []byte) []byte { return append(uleb(uint64(len(b))), b...) }
	alloc := []byte{0x01, 0x01, 0x7f, 0x23, 0x00, 0x21, 0x01, 0x20, 0x01, 0x23, 0x00, 0x41, 0x0c, 0x6a, 0x36, 0x02, 0x00, 0x20, 0x01, 0x20, 0x00, 0x36, 0x02, 0x04, 0x20, 0x01, 0x41, 0x00, 0x36, 0x02, 0x08, 0x23, 0x00, 0x41, 0x0c, 0x6a, 0x20, 0x00, 0x6a, 0x24, 0x00, 0x20, 0x01, 0x0b}
	bodies := [][]byte{wrap(alloc), wrap([]byte{0x00, 0x0b}), wrap([]byte{0x00, 0x0b}), wrap([]byte{0x00, 0x41, 0x00, 0x0b})}
	for i := 0; i < n; i++ {
		bodies = append(bodies, wrap(append(append(append([]byte{}, locals...), body...), 0x0b)))
	}
	mod := []byte{0x00, 'a', 's', 'm', 0x01, 0x00, 0x00, 0x00}
	for _, s := range [][]byte{section(1, typeSec), section(3, ccVec(funcs...)), section(4, tableSec), section(5, memSec), section(6, globSec), section(7, expSec), section(10, ccVec(bodies...))} {
		mod = append(mod, s...)
	}
	return mod
}

// ccModuleTypes is ccModule with extra (unused) function types declared
// after its own.
func ccModuleTypes(extra [][]byte) []byte {
	m := ccModule(0, nil, nil)
	r := &reader{b: m, pos: 8}
	_, _ = r.byte()
	n, _ := r.u32()
	typeEnd := r.pos + int(n)
	types := [][]byte{{0x60, 1, 0x7f, 1, 0x7f}, {0x60, 1, 0x7f, 0}, {0x60, 0, 0}, {0x60, 3, 0x7f, 0x7f, 0x7f, 1, 0x7f}}
	out := append([]byte{}, m[:8]...)
	out = append(out, section(1, ccVec(append(types, extra...)...))...)
	return append(out, m[typeEnd:]...)
}

func ccDir(root string) int64 {
	var n int64
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.Contains(p, "modules") {
			n += info.Size()
		}
		return nil
	})
	return n
}

func ccRead(f string) []byte {
	bz, _ := os.ReadFile(f)
	if len(bz) > 2 && bz[0] == 0x1f && bz[1] == 0x8b {
		zr, _ := gzip.NewReader(bytes.NewReader(bz))
		bz, _ = io.ReadAll(zr)
	}
	return bz
}

type ccRes struct {
	name             string
	size, fns        int
	real, est, units uint64
	load             time.Duration
}

func ccMeasure(t *testing.T, name string, code []byte, timeLoad bool) (ccRes, bool) {
	st, err := Inspect(code)
	if err != nil {
		return ccRes{}, false
	}
	dir := t.TempDir()
	vm, err := wasmvm.NewVM(dir, []string{"iterator", "staking", "stargate", "cosmwasm_1_1", "cosmwasm_1_2", "cosmwasm_1_3", "cosmwasm_1_4", "cosmwasm_2_0", "cosmwasm_2_1", "cosmwasm_2_2"}, 32, false, 100)
	if err != nil {
		t.Fatal(err)
	}
	cs, _, err := vm.StoreCode(code, 1<<62)
	vm.Cleanup()
	if err != nil {
		return ccRes{}, false
	}
	real := uint64(ccDir(dir))
	var load time.Duration
	if timeLoad {
		vm2, _ := wasmvm.NewVM(dir, []string{"iterator", "staking", "stargate", "cosmwasm_1_1", "cosmwasm_1_2", "cosmwasm_1_3", "cosmwasm_1_4", "cosmwasm_2_0", "cosmwasm_2_1", "cosmwasm_2_2"}, 32, false, 100)
		t0 := time.Now()
		_ = vm2.Pin(cs)
		load = time.Since(t0)
		m, _ := vm2.GetMetrics()
		if m.SizePinnedMemoryCache > real {
			real = m.SizePinnedMemoryCache
		}
		vm2.Cleanup()
	}
	est := ConservativeCompiledSize(len(code), st)
	return ccRes{name, len(code), st.Functions, real, est, 0, load}, true
}

// The VM compiles for the machine it runs on, and validators run x86-64:
// CI must run this test on x86-64 as well as arm64 (on Apple silicon:
// GOARCH=amd64 CC="clang -arch x86_64" under Rosetta; libwasmvm.dylib is
// universal).
//
// The conservative compiled size is at or above what the VM really keeps
// (on disk and in its memory cache) for the release contracts and for
// shapes an uploader chooses; the shapes that compile largest are over the
// upload bound and refused.
func TestConservativeCompiledSizeCoversCompiled(t *testing.T) {
	const target = 790 * 1024
	shape := func(n int, locals, pat []byte) []byte {
		per := target/n - len(locals) - 4
		reps := per / len(pat)
		if reps < 1 {
			reps = 1
		}
		return ccModule(n, locals, ccRep(reps, pat...))
	}
	cases := map[string][]byte{
		"hackatom":      wasmtestdata.HackatomContractWasm(),
		"nops":          shape(4096, []byte{0x00}, []byte{0x01}),
		"call_indirect": shape(4096, []byte{0x00}, []byte{0x41, 0x00, 0x11, 0x02, 0x00}),
		"call":          shape(4096, []byte{0x00}, []byte{0x10, 0x02}),
		"i64.div_s":     shape(4096, []byte{0x01, 0x02, 0x7e}, []byte{0x20, 0x00, 0x20, 0x01, 0x7f, 0x1a}),
		"block/br_if":   shape(4096, []byte{0x00}, []byte{0x02, 0x40, 0x41, 0x00, 0x0d, 0x00, 0x0b}),
		"loop":          shape(4096, []byte{0x00}, []byte{0x03, 0x40, 0x0b}),
		"if":            shape(4096, []byte{0x00}, []byte{0x41, 0x00, 0x04, 0x40, 0x0b}),
	}
	none := []byte{0x00}
	loc := func(n uint64, typ byte) []byte { return append(append([]byte{0x01}, uleb(n)...), typ) }
	i32c0 := []byte{0x41, 0x00}
	chain := func(prefix, pat []byte) []byte {
		per := target/512 - len(prefix) - 1 - 4
		return ccModule(512, none, append(append(append([]byte{}, prefix...), ccRep(per/len(pat), pat...)...), 0x1a))
	}
	brTable := func(fns, targets int) []byte {
		b := []byte{0x02, 0x40, 0x20, 0x00, 0x0e}
		b = append(b, uleb(uint64(targets))...)
		b = append(b, ccRep(targets+1, 0x00)...)
		b = append(b, 0x0b)
		per := target/fns - 7
		return ccModule(fns, loc(1, 0x7f), ccRep(max(1, per/len(b)), b...))
	}
	// Calibration shapes: the store constant forms, bit counts, loads, the
	// control forms that compile largest, br_table targets and the locals
	// series (small modules where the per-module base dominates).
	cases["i64.store const"] = shape(4096, none, []byte{0x41, 0x00, 0x42, 0x00, 0x37, 0x03, 0x00})
	cases["i32.store const"] = shape(4096, none, []byte{0x41, 0x00, 0x41, 0x00, 0x36, 0x02, 0x00})
	cases["i64.store16 const"] = shape(4096, none, []byte{0x41, 0x00, 0x42, 0x00, 0x3D, 0x01, 0x00})
	cases["i32.clz chain"] = chain(i32c0, []byte{0x67})
	cases["i64.clz chain"] = chain([]byte{0x42, 0x00}, []byte{0x79})
	cases["i32.load8_u chain"] = chain(i32c0, []byte{0x2D, 0x00, 0x00})
	cases["i64.rem_s const"] = chain([]byte{0x42, 0x00}, []byte{0x42, 0x7f, 0x81})
	cases["if/else const"] = shape(512, none, []byte{0x41, 0x00, 0x04, 0x40, 0x05, 0x0B})
	cases["nest loops"] = ccModule(100, none, append(ccRep(3000, 0x03, 0x40), ccRep(3000, 0x0B)...))
	cases["br_table 64"] = brTable(512, 64)
	cases["br_table 4096"] = brTable(160, 4096)
	for _, n := range []int{24, 48, 72, 96} {
		cases[fmt.Sprintf("%d fns of 100 i64 locals", n)] = ccModule(n, loc(100, 0x7e), nil)
	}
	for _, l := range []uint64{0, 25, 50, 100} {
		cases[fmt.Sprintf("68 fns of %d locals", l)] = ccModule(68, loc(l, 0x7e), nil)
	}
	// Straight-line code that keeps many values on the value stack: past
	// the registers every operand and result goes through memory, and a
	// call saves every live value (round 13). Depth is the values kept
	// below the chain.
	deep := func(depth int, op []byte) []byte {
		const fns, size = 16, 200 * 1024
		b := ccRep(depth+1, 0x20, 0x00)
		b = append(b, ccRep((size/fns-2*depth-24)/len(op), op...)...)
		return ccModule(fns, loc(2, 0x7e), append(b, ccRep(depth+1, 0x1A)...))
	}
	for _, d := range []int{0, 3, 4, 32} {
		cases[fmt.Sprintf("memory.size+grow;ext;add d%d", d)] = deep(d, []byte{0x3f, 0x00, 0x40, 0x00, 0xac, 0x7c})
		cases[fmt.Sprintf("memory.size;drop d%d", d)] = deep(d, []byte{0x3f, 0x00, 0x1a})
		cases[fmt.Sprintf("call d%d", d)] = deep(d, []byte{0x10, 0x02})
		cases[fmt.Sprintf("call_indirect d%d", d)] = deep(d, []byte{0x41, 0x00, 0x11, 0x02, 0x00})
		cases[fmt.Sprintf("loop d%d", d)] = deep(d, []byte{0x03, 0x40, 0x0b})
		cases[fmt.Sprintf("if/else d%d", d)] = deep(d, []byte{0x41, 0x00, 0x04, 0x40, 0x05, 0x0b})
		cases[fmt.Sprintf("block/br_if d%d", d)] = deep(d, []byte{0x02, 0x40, 0x41, 0x00, 0x0d, 0x00, 0x0b})
		cases[fmt.Sprintf("i64.load offset d%d", d)] = deep(d, []byte{0xa7, 0x29, 0x03, 0xff, 0xff, 0xff, 0xff, 0x0f})
		cases[fmt.Sprintf("i64.eqz;ext d%d", d)] = deep(d, []byte{0x50, 0xad})
		cases[fmt.Sprintf("i64.extend32_s d%d", d)] = deep(d, []byte{0xc4})
		cases[fmt.Sprintf("i64.clz d%d", d)] = deep(d, []byte{0x79})
		cases[fmt.Sprintf("i64.rem_s d%d", d)] = deep(d, []byte{0x20, 0x01, 0x81})
		cases[fmt.Sprintf("global.get/add d%d", d)] = deep(d, []byte{0x23, 0x00, 0xad, 0x7c})
	}
	// The review's accepted 790 KiB upload of this shape compiled to 44.5
	// MB, 3.2 times the estimate then: now over the upload bound.
	cases["memory.size+grow;ext;add d40 790 KiB"] = func() []byte {
		b := ccRep(41, 0x20, 0x00)
		b = append(b, ccRep((790*1024/64-120)/6, 0x3f, 0x00, 0x40, 0x00, 0xac, 0x7c)...)
		return ccModule(64, loc(2, 0x7e), append(b, ccRep(41, 0x1A)...))
	}()
	// Many distinct signatures: the VM makes trampolines for each.
	for _, np := range []int{4, 100} {
		var types [][]byte
		for k := 0; len(types)*(np+3) < 200*1024; k++ {
			p := ccRep(np, 0x7e)
			p[k%np], p[(k/np)%np], p[(k/(np*np))%np] = 0x7f, 0x7d, 0x7c
			types = append(types, append(append([]byte{0x60}, uleb(uint64(np))...), append(p, 0x01, 0x7e)...))
		}
		cases[fmt.Sprintf("%d types of %d params", len(types), np)] = ccModuleTypes(types)
	}
	for name, code := range map[string][]byte{
		"reflect_2_0": wasmtestdata.ReflectContractWasm(),
		"reflect_1_1": wasmtestdata.MigrateReflectContractWasm(),
		"cyberpunk":   wasmtestdata.CyberpunkContractWasm(),
		"ibc_reflect": wasmtestdata.IBCReflectContractWasm(),
		"burner":      wasmtestdata.BurnerContractWasm(),
	} {
		cases[name] = code
	}
	rel, _ := filepath.Glob(os.Getenv("QORE_HANDLER_ROOTS") + "/app/testdata/*.wasm.gz")
	for _, f := range rel {
		cases[filepath.Base(f)] = ccRead(f)
	}
	for name, code := range cases {
		r, ok := ccMeasure(t, name, code, true)
		if !ok {
			t.Fatalf("%s: refused", name)
		}
		t.Logf("%-28s wasm %7d, compiled %9d, estimate %9d (%.2f)%s", name, r.size, r.real, r.est, float64(r.real)/float64(r.est),
			map[bool]string{true: ", over the upload bound", false: ""}[r.est > MaxUploadCompiledSize])
		if r.real > r.est {
			t.Errorf("%s: compiled %d is above the estimate %d", name, r.real, r.est)
		}
		if strings.HasSuffix(name, ".wasm.gz") && r.est > MaxUploadCompiledSize {
			t.Errorf("%s: a release contract is over the upload bound", name)
		}
	}
}
