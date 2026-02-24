package types

// SVMExecutor abstracts the BPF execution engine (Rust FFI in proprietary, stub in public).
type SVMExecutor interface {
	// Execute runs a BPF program with the given instruction and accounts.
	Execute(program []byte, instruction []byte, accounts []SVMAccount,
		computeBudget uint64) (*ExecutionResult, error)

	// ValidateProgram verifies a BPF ELF binary is well-formed.
	ValidateProgram(bytecode []byte) error

	// Close releases executor resources.
	Close()
}
