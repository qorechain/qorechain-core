package app

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

type homeOpts map[string]interface{}

func (o homeOpts) Get(key string) interface{} { return o[key] }

// The x/wasm VM directory follows --home. A node whose code already lives in
// the default home's directory, and that has none under its own home, keeps
// it, because that directory is its only copy of the contract bytecode.
func TestWasmDataDirFollowsHome(t *testing.T) {
	prev := DefaultNodeHome
	t.Cleanup(func() { DefaultNodeHome = prev })
	root := t.TempDir()
	DefaultNodeHome = filepath.Join(root, "default")
	legacy := filepath.Join(DefaultNodeHome, "wasm")
	nodeB := filepath.Join(root, "nodeB")

	// No --home (an in-process app): a private directory, never the
	// default home's.
	if got := wasmDataDir(homeOpts{}); got == legacy || filepath.Dir(got) == DefaultNodeHome || !dirExists(got) {
		t.Fatalf("no --home: %s", got)
	} else {
		_ = os.RemoveAll(got)
	}
	if got := wasmDataDir(homeOpts{flags.FlagHome: DefaultNodeHome}); got != legacy {
		t.Fatalf("--home is the default: %s", got)
	}
	// No legacy directory: the node's own.
	if got := wasmDataDir(homeOpts{flags.FlagHome: nodeB}); got != filepath.Join(nodeB, "wasm") {
		t.Fatalf("fresh node: %s", got)
	}
	// Legacy directory present, but a new node (no chain data): its own.
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := wasmDataDir(homeOpts{flags.FlagHome: nodeB}); got != filepath.Join(nodeB, "wasm") {
		t.Fatalf("new node next to a legacy directory: %s", got)
	}
	// An existing node with its code in the legacy directory: keep it.
	if err := os.MkdirAll(filepath.Join(nodeB, "data", "application.db"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := wasmDataDir(homeOpts{flags.FlagHome: nodeB}); got != legacy {
		t.Fatalf("existing node: %s, want the legacy %s", got, legacy)
	}
	// Once the operator moved it under --home, that is used.
	if err := os.MkdirAll(filepath.Join(nodeB, "wasm"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := wasmDataDir(homeOpts{flags.FlagHome: nodeB}); got != filepath.Join(nodeB, "wasm") {
		t.Fatalf("moved node: %s", got)
	}
}

// The [wasm] section is honoured, and the simulation bound is always set.
func TestReadWasmNodeConfig(t *testing.T) {
	cfg := readWasmNodeConfig(homeOpts{"wasm.query_gas_limit": uint64(1_234_567), "wasm.memory_cache_size": uint32(256)})
	if cfg.SmartQueryGasLimit != 1_234_567 || cfg.MemoryCacheSize != 256 {
		t.Fatalf("the [wasm] section was not read: %+v", cfg)
	}
	if cfg.SimulationGasLimit == nil || *cfg.SimulationGasLimit != DefaultWasmSimulationGasLimit {
		t.Fatalf("simulation limit %v, want the default", cfg.SimulationGasLimit)
	}
	cfg = readWasmNodeConfig(homeOpts{"wasm.simulation_gas_limit": "20000000"})
	if *cfg.SimulationGasLimit != 20_000_000 {
		t.Fatalf("the operator's simulation limit was not honoured: %d", *cfg.SimulationGasLimit)
	}
}

// The limit a node runs with: the operator's value when it is a real bound,
// the default otherwise.
func TestBoundedSimulationGasLimit(t *testing.T) {
	v := func(n uint64) *uint64 { return &n }
	for name, tc := range map[string]struct {
		cfg  *wasmtypes.NodeConfig
		want uint64
	}{
		"no config":      {nil, DefaultWasmSimulationGasLimit},
		"key absent":     {&wasmtypes.NodeConfig{}, DefaultWasmSimulationGasLimit},
		"zero":           {&wasmtypes.NodeConfig{SimulationGasLimit: v(0)}, DefaultWasmSimulationGasLimit},
		"infinite":       {&wasmtypes.NodeConfig{SimulationGasLimit: v(math.MaxUint64)}, DefaultWasmSimulationGasLimit},
		"operator lower": {&wasmtypes.NodeConfig{SimulationGasLimit: v(10_000_000)}, 10_000_000},
		"operator high":  {&wasmtypes.NodeConfig{SimulationGasLimit: v(200_000_000)}, 200_000_000},
	} {
		if got := *boundedSimulationGasLimit(tc.cfg); got != tc.want {
			t.Errorf("%s: limit %d, want %d", name, got, tc.want)
		}
	}
}

// A node runs with at least the minimum memory cache: a smaller configured
// value, or none, is raised to it.
func TestWasmMemoryCacheMinimum(t *testing.T) {
	if cfg := readWasmNodeConfig(homeOpts{"wasm.memory_cache_size": uint32(100)}); cfg.MemoryCacheSize != MinWasmMemoryCacheMiB {
		t.Fatalf("100 MiB configured: running with %d", cfg.MemoryCacheSize)
	}
	if cfg := readWasmNodeConfig(homeOpts{}); cfg.MemoryCacheSize != MinWasmMemoryCacheMiB {
		t.Fatalf("none configured: running with %d", cfg.MemoryCacheSize)
	}
	if err := checkWasmMemoryCache(wasmtypes.NodeConfig{MemoryCacheSize: 7}); err == nil {
		t.Fatal("7 MiB reported as enough")
	}
}
