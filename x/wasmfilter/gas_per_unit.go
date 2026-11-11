package wasmfilter

// Every charge of the block wasm budget costs at least one gas per budget
// unit: contract execution (one SDK gas of VM work at the base price is one
// unit), code loading (LoadGasPerUnit), call setup and memory growth
// (CallSetupGasPerUnit), repeated loads (the same fraction of both), code
// upload (CompileGasPerByte + UploadStorageGasPerByte + UploadInspectGasPerByte
// per byte unit, one gas per CompiledBytesPerGas compiled bytes against one
// unit per CompiledBytesPerUploadUnit) and async calls (their prepay holds
// the load and setup gas with the units), and the congestion factor only
// raises the gas.
//
// The invariant is load-bearing, not only an economic choice: a budget
// abort raised inside a nested smart query reaches the VM as an ordinary
// query error, and the transaction still fails because the gas the query
// was given cannot cover units the budget has refused. Breaking it would
// let a contract catch a budget abort. The constant expressions below stop
// the build if a retuned constant breaks it; the app tests measure it on
// every charge.
const (
	_ = uint64(LoadGasPerUnit - 1)
	_ = uint64(CallSetupGasPerUnit - 1)
	_ = uint64(CompileGasPerByte + UploadStorageGasPerByte + UploadInspectGasPerByte - 1)
	_ = uint64(CompiledBytesPerUploadUnit/CompiledBytesPerGas - 1)
)
