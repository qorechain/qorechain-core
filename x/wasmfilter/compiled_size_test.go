package wasmfilter

import (
	"bytes"
	"compress/gzip"
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
