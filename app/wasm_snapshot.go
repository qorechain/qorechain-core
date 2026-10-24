package app

import (
	"fmt"

	"cosmossdk.io/store/snapshots"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

// registerWasmSnapshotter adds the CosmWasm extension to the node's state-sync
// snapshots. Contract code lives in the node's wasm directory, not in the
// multistore, so without it a node restored from a snapshot has every code's
// metadata and none of its bytes: the first execution of an existing contract
// then fails on that node alone and its block result differs from the rest.
// Node-local: the extension changes what snapshots carry, not block execution.
// A snapshot that carries it can only be restored by a binary that registers
// it, which every node past the v3.2.5 height runs.
func (app *QoreChainApp) registerWasmSnapshotter(manager *snapshots.Manager) error {
	if manager == nil {
		return nil
	}
	if err := manager.RegisterExtensions(wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmKeeper)); err != nil {
		return fmt.Errorf("register wasm snapshot extension: %w", err)
	}
	return nil
}
