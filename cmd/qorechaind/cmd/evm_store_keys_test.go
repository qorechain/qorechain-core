package cmd

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
)

// QSR-2026-0520: the EVM keeper builds the StateDB snapshot store from its key
// map, so that map must cover every KV store the app mounts, the hand-mounted
// ones included. Before the fix "evm" itself was missing and every stateful
// precompile call panicked.
func TestEVMKeeperHasEveryMountedStore(t *testing.T) {
	a := sharedTestApp(t)
	keys := a.EVMKeeper.KVStoreKeys()
	mounted := a.CommitMultiStore().(interface {
		StoreKeysByName() map[string]storetypes.StoreKey
	}).StoreKeysByName()

	n := 0
	for name, k := range mounted {
		kv, ok := k.(*storetypes.KVStoreKey)
		if !ok {
			continue
		}
		n++
		if keys[name] != kv {
			t.Errorf("mounted KV store %q is missing from the EVM keeper's keys", name)
		}
	}
	for _, name := range []string{"evm", "feemarket", "erc20", "precisebank", "ibc", "transfer", "wasm", "pqc", "crossvm", "bank", "acc"} {
		if keys[name] == nil {
			t.Errorf("store %q is missing from the EVM keeper's keys", name)
		}
	}
	if n < 20 {
		t.Fatalf("only %d KV stores mounted; the app was not built as expected", n)
	}
}
