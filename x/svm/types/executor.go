package types

// SVMExecutor abstracts the BPF execution engine (Rust FFI in proprietary, stub in public).
type SVMExecutor interface {
	// Execute runs a BPF program with the given instruction and accounts.
	Execute(program []byte, instruction []byte, accounts []SVMAccount,
		computeBudget uint64) (*ExecutionResult, error)

	// ExecuteV2 runs a BPF program with full Solana-compatible account context.
	// Accounts are serialized into the BPF input format and modified accounts
	// are deserialized from the result buffer after execution.
	ExecuteV2(program []byte, accounts []SVMAccount, metas []AccountMeta,
		instructionData []byte, programID [32]byte,
		computeBudget uint64) (*ExecutionResult, error)

	// ExecuteNative runs a native program directly (no BPF interpretation).
	// The program is identified by its 32-byte ID and must be registered in
	// the runtime's native program table.
	ExecuteNative(programID [32]byte, accounts []SVMAccount, metas []AccountMeta,
		instructionData []byte) (*ExecutionResult, error)

	// ValidateProgram verifies a BPF ELF binary is well-formed.
	ValidateProgram(bytecode []byte) error

	// Close releases executor resources.
	Close()
}
