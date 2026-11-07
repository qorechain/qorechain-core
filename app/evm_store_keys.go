package app

import (
	"fmt"
	"sort"

	storetypes "cosmossdk.io/store/types"
)

// completeEVMStoreKeys gives the EVM keeper every KV store the application
// mounts (QSR-2026-0520).
//
// The EVM keeper takes a map of store keys and builds the StateDB's snapshot
// store from it whenever a stateful precompile runs. That map came from the
// runtime's store keys, which hold only the stores registered through
// dependency injection; every store mounted by hand in NewQoreChainApp (evm,
// feemarket, precisebank, erc20, ibc, transfer, wasm and all QoreChain modules)
// was missing from it, so the first stateful precompile call to touch one
// panicked with "kv store with key evm has not been registered in stores".
//
// The keeper keeps a reference to the map it was given, so filling the same
// map once every store is mounted is enough. Called right before Load, after
// the last MountStores.
//
// The map is read whenever a stateful precompile runs. The standard
// cosmos/evm precompiles (0x08xx) are not active on any network
// (active_static_precompiles is empty, there are no ERC-20 token pairs), but
// QoreChain's own precompiles are wired outside that list and switched on by
// protocol features: on testnet the CrossVM bridge (0x0901), PQC verify
// (0x0A01, 0x0A02) and RL consensus params (0x0C01) answer, and they run on
// this completed map. Mainnet has none of them active.
func (app *QoreChainApp) completeEVMStoreKeys() {
	if app.EVMKeeper == nil {
		return
	}
	mounted, ok := app.CommitMultiStore().(interface {
		StoreKeysByName() map[string]storetypes.StoreKey
	})
	if !ok {
		panic("the commit multistore cannot list its mounted stores")
	}
	keys := app.EVMKeeper.KVStoreKeys()
	if keys == nil {
		panic("the EVM keeper holds no store-key map")
	}
	fillStoreKeyMap(keys, mounted.StoreKeysByName())
	if missing := missingStoreKeys(keys, mounted.StoreKeysByName()); len(missing) > 0 {
		panic(fmt.Sprintf("mounted KV stores missing from the EVM keeper's store keys: %v", missing))
	}
}

// fillStoreKeyMap adds every KV store key in mounted to keys, by name.
func fillStoreKeyMap(keys map[string]*storetypes.KVStoreKey, mounted map[string]storetypes.StoreKey) {
	for name, k := range mounted {
		if kv, ok := k.(*storetypes.KVStoreKey); ok {
			if _, have := keys[name]; !have {
				keys[name] = kv
			}
		}
	}
}

// missingStoreKeys lists the mounted KV stores that keys lacks, or that it
// holds under a different key instance, sorted by name.
func missingStoreKeys(keys map[string]*storetypes.KVStoreKey, mounted map[string]storetypes.StoreKey) []string {
	var missing []string
	for name, k := range mounted {
		kv, ok := k.(*storetypes.KVStoreKey)
		if !ok {
			continue
		}
		if have, found := keys[name]; !found || have != kv {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}
