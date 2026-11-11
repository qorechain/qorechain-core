package app

import (
	"fmt"

	runtimev1alpha1 "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"

	"github.com/cosmos/cosmos-sdk/runtime"
)

// assertRuntimeKeepsWasmHooks refuses to start when the runtime module
// config would replace the PrepareCheckStater the application sets for the
// wasm gas multiplier (wasm_gas_register.go): the runtime sets its own
// whenever PrepareCheckStaters are configured, which would silently leave
// CheckTx and simulations priced at a stale congestion factor. The
// PreBlocker is safe (the runtime keeps one that is already set).
func assertRuntimeKeepsWasmHooks() {
	for _, mc := range ModuleConfig {
		if mc.Name != runtime.ModuleName || mc.Config == nil {
			continue
		}
		var m runtimev1alpha1.Module
		if err := mc.Config.UnmarshalTo(&m); err != nil {
			panic(fmt.Sprintf("runtime module config: %v", err))
		}
		if len(m.PrepareCheckStaters) != 0 {
			panic("runtime PrepareCheckStaters are configured: they would replace the application's " +
				"post-commit wasm gas multiplier sync; chain the module PrepareCheckStaters into " +
				"syncWasmGasMultiplier's hook before adding them")
		}
	}
}
