package types

const (
	// FeatureWasmCodeFilter puts a deterministic bytecode check (x/wasmfilter)
	// in front of the CosmWasm VM. From this switch a message that stores code
	// (MsgStoreCode, MsgStoreAndInstantiateContract, MsgStoreAndMigrateContract,
	// sent by an account or executed by governance) is refused when its
	// bytecode fails the check, and so is every call that would run a stored
	// code that fails it: instantiate, execute, migrate, sudo, reply, smart
	// queries and the IBC entry points. The check refuses block types with a
	// type-index (multi-value) signature, br_table with more than 4,096
	// targets, opcodes outside the CosmWasm feature set and any module it
	// cannot fully parse. Restoring code bytes (genesis import, state-sync
	// snapshots) and pinning are never refused. Code that passes the check runs
	// exactly as before, with the same gas. Below the switch nothing is
	// checked.
	FeatureWasmCodeFilter = "wasm-code-filter"
)

// V328WasmFilterFeatures are the v3.2.8 switches for the CosmWasm code filter.
var V328WasmFilterFeatures = []string{
	FeatureWasmCodeFilter,
}
