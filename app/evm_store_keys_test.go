package app

import (
	"reflect"
	"testing"

	storetypes "cosmossdk.io/store/types"
)

// QSR-2026-0520: a store mounted by hand must reach the EVM keeper's key map,
// and the assertion must name any store that does not.
func TestFillStoreKeyMapAddsHandMountedStores(t *testing.T) {
	bank := storetypes.NewKVStoreKey("bank")
	evm := storetypes.NewKVStoreKey("evm")
	wasm := storetypes.NewKVStoreKey("wasm")
	mounted := map[string]storetypes.StoreKey{
		"bank":           bank,
		"evm":            evm,
		"wasm":           wasm,
		"transient_evm":  storetypes.NewTransientStoreKey("transient_evm"),
		"memory_capable": storetypes.NewMemoryStoreKey("memory_capable"),
	}
	// What the runtime hands over: only the dependency-injected stores.
	keys := map[string]*storetypes.KVStoreKey{"bank": bank}

	if got := missingStoreKeys(keys, mounted); !reflect.DeepEqual(got, []string{"evm", "wasm"}) {
		t.Fatalf("before the fill, missing = %v, want [evm wasm]", got)
	}
	fillStoreKeyMap(keys, mounted)
	if got := missingStoreKeys(keys, mounted); len(got) != 0 {
		t.Fatalf("after the fill, still missing %v", got)
	}
	if keys["evm"] != evm || keys["wasm"] != wasm || len(keys) != 3 {
		t.Fatalf("unexpected key map after fill: %v", keys)
	}

	// A different key instance under the same name is a wiring error too.
	keys["wasm"] = storetypes.NewKVStoreKey("wasm")
	if got := missingStoreKeys(keys, mounted); !reflect.DeepEqual(got, []string{"wasm"}) {
		t.Fatalf("a foreign key instance was not reported: %v", got)
	}
}
