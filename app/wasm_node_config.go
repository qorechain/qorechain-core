package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	wasm "github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cast"
)

// DefaultWasmSimulationGasLimit bounds the gas of one transaction simulation
// when the node's app.toml sets no [wasm] simulation_gas_limit. Without a
// bound, and with a block max_gas of -1 (the value on both live networks),
// wasmd leaves the simulation on the infinite gas meter the SDK gives it, and
// the VM then runs a contract with a gas limit of 2^64-1: one unauthenticated
// /cosmos/tx/v1beta1/simulate request against a contract that loops would pin
// a CPU core for as long as the process lives. 50M is far above any real
// transaction. Node-local: it bounds simulation only, block execution is
// untouched.
const DefaultWasmSimulationGasLimit uint64 = 50_000_000

// boundedSimulationGasLimit is the simulation gas limit a node runs with: the
// operator's [wasm] simulation_gas_limit when it is a real bound, and
// DefaultWasmSimulationGasLimit when it is absent (any app.toml written before
// the template carried the key, or a node started with flags only), zero
// (which wasmd refuses at startup) or 2^64-1 (an infinite meter by another
// name). It is never nil, so wasmd's decorator never falls back to the block
// max_gas, which is -1 on the live networks.
func boundedSimulationGasLimit(cfg *wasmtypes.NodeConfig) *uint64 {
	limit := DefaultWasmSimulationGasLimit
	if cfg != nil && cfg.SimulationGasLimit != nil {
		if v := *cfg.SimulationGasLimit; v != 0 && v != math.MaxUint64 {
			limit = v
		}
	}
	return &limit
}

// readWasmNodeConfig reads the node's [wasm] app.toml section (query gas
// limit, memory cache, simulation gas limit; --trace turns on contract debug
// output) with the simulation bound always set. The x/wasm keeper and the
// ante handler are built from this one value. All of it is node-local.
func readWasmNodeConfig(appOpts servertypes.AppOptions) wasmtypes.NodeConfig {
	cfg, err := wasm.ReadNodeConfig(appOpts)
	if err != nil {
		panic("failed to read wasm node config: " + err.Error())
	}
	cfg.SimulationGasLimit = boundedSimulationGasLimit(&cfg)
	return cfg
}

// wasmDataDir is where the x/wasm VM keeps contract code and caches: the
// "wasm" directory under the node's --home. Before, it was always under the
// default home (~/.qorechaind), so two nodes run by one user with different
// --home shared one VM directory and its lock file.
//
// An existing node (one with application state under its --home) whose code is in
// the default home's directory, and which has none under its own --home, keeps
// using that directory: the VM directory is the only copy of the contract
// bytecode outside a state-sync snapshot, and a node that lost it would fail
// every contract call. To move such a node, stop it and move (or copy) that
// directory to <home>/wasm; from then on <home>/wasm is used. A new node
// (no application state yet) always gets <home>/wasm.
func wasmDataDir(appOpts servertypes.AppOptions) string {
	legacy := filepath.Join(DefaultNodeHome, "wasm")
	home := ""
	if appOpts != nil {
		home = cast.ToString(appOpts.Get(flags.FlagHome))
	}
	if home == "" {
		return legacy
	}
	own := filepath.Join(home, "wasm")
	existingNode := dirExists(filepath.Join(home, "data", "application.db"))
	if filepath.Clean(own) == filepath.Clean(legacy) || dirExists(own) || !existingNode || !dirExists(legacy) {
		return own
	}
	return legacy
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// checkWasmCodePresent refuses to run a node whose x/wasm VM directory holds
// none of the contract code the chain has stored. Such a node would fail every
// contract call, compute different results from the rest of the network at
// the first one and halt there; refusing at startup says why instead. It
// stops at the first code it finds, so a healthy node reads one file. A chain
// with no stored code (a new chain, a node about to state-sync, whose
// snapshot restores the code) passes.
func (app *QoreChainApp) checkWasmCodePresent(ctx sdk.Context, dir string) error {
	stored, present := 0, false
	app.WasmKeeper.IterateCodeInfos(ctx, func(id uint64, _ wasmtypes.CodeInfo) bool {
		stored++
		if bz, err := app.WasmKeeper.GetByteCode(ctx, id); err == nil && len(bz) > 0 {
			present = true
			return true
		}
		return false
	})
	if stored == 0 || present {
		return nil
	}
	return fmt.Errorf("the chain has %d stored CosmWasm codes but the wasm directory %s holds none of them; "+
		"if this node's code is in another wasm directory (the default home's is %s), stop the node and "+
		"move or copy it to %s, or restore it from a state-sync snapshot", stored, dir,
		filepath.Join(DefaultNodeHome, "wasm"), dir)
}
