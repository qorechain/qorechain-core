package types

// SVMExecutor abstracts the BPF execution engine (Rust FFI in full, stub in public).
type SVMExecutor interface {
	// Execute runs a BPF program with the given instruction and accounts.
	Execute(program []byte, instruction []byte, accounts []SVMAccount,
		computeBudget uint64) (*ExecutionResult, error)

	// ExecuteV2 runs a BPF program with full Solana-compatible account context.
	// Accounts are serialized into the BPF input format and modified accounts
	// are deserialized from the result buffer after execution.
	ExecuteV2(program []byte, accounts []SVMAccount, metas []AccountMeta,
		instructionData []byte, programID [32]byte,
		computeBudget uint64, blockTime int64) (*ExecutionResult, error)

	// ExecuteV2WithPrograms runs a BPF program like ExecuteV2 but also supplies a
	// registry of other deployed programs (keyed by 32-byte program ID), enabling
	// BPF->BPF cross-program invocation from the Solana-compatible execution path.
	ExecuteV2WithPrograms(program []byte, accounts []SVMAccount, metas []AccountMeta,
		instructionData []byte, programID [32]byte, programs map[[32]byte][]byte,
		computeBudget uint64, blockTime int64) (*ExecutionResult, error)

	// ExecuteNative runs a native program directly (no BPF interpretation).
	// The program is identified by its 32-byte ID and must be registered in
	// the runtime's native program table.
	ExecuteNative(programID [32]byte, accounts []SVMAccount, metas []AccountMeta,
		instructionData []byte, blockTime int64) (*ExecutionResult, error)

	// ValidateProgram verifies a BPF ELF binary is well-formed.
	ValidateProgram(bytecode []byte) error

	// Close releases executor resources.
	Close()
}

// MeteredExecutor is implemented by a runtime that can charge syscalls by the
// bytes they touch (v3.2.5, svm-compute-gas). Once that feature is active the
// keeper runs BPF programs only through it: without byte metering the compute
// unit count does not bound the work a program does, so charging gas per unit
// would not bound it either.
type MeteredExecutor interface {
	ExecuteV2Metered(program []byte, accounts []SVMAccount, metas []AccountMeta,
		instructionData []byte, programID [32]byte, programs map[[32]byte][]byte,
		computeBudget uint64, blockTime int64) (*ExecutionResult, error)
}

// Runtime flags of the v3 entry (v3.2.8). Each mirrors a FLAG_* constant of
// the Rust runtime and is set from the protocol feature named beside it.
const (
	RuntimeFlagByteMetering     uint64 = 1 << 0 // svm-compute-gas (v3.2.5)
	RuntimeFlagFullResult       uint64 = 1 << 1 // svm-result-full-fields
	RuntimeFlagTokenOwnerCheck  uint64 = 1 << 2 // svm-token-owner-checks
	RuntimeFlagCPIDepthBounded  uint64 = 1 << 3 // svm-cpi-depth-bounded
	RuntimeFlagMemcpyNoOverlap  uint64 = 1 << 4 // svm-memcpy-no-overlap
	RuntimeFlagPDASeedLimits    uint64 = 1 << 5 // svm-pda-seed-limits
	RuntimeFlagConsensusSysvars uint64 = 1 << 6 // svm-consensus-sysvars
)

// RuntimeSysvars are the consensus values of the Clock and Rent sysvars handed
// to the runtime with RuntimeFlagConsensusSysvars. ExemptionThresholdBits is the
// IEEE 754 encoding of the rent exemption threshold, computed by the keeper from
// the decimal param with a correctly rounded parse, so every node passes the
// same bits.
type RuntimeSysvars struct {
	Slot                   uint64
	Epoch                  uint64
	LeaderScheduleEpoch    uint64
	EpochStartTimestamp    int64
	LamportsPerByteYear    uint64
	ExemptionThresholdBits uint64
	BurnPercent            uint8
}

// RuntimeOptions are the arguments of the v3 entry beyond the v2 ones.
type RuntimeOptions struct {
	Flags   uint64
	Sysvars RuntimeSysvars
}

// HardenedExecutor is implemented by a runtime that has the v3 entry
// (v3.2.8). Once any v3.2.8 runtime switch is active the keeper runs native
// and user programs only through it; a runtime without it is refused rather
// than run without the switches the chain asked for.
type HardenedExecutor interface {
	ExecuteV3(program []byte, accounts []SVMAccount, metas []AccountMeta,
		instructionData []byte, programID [32]byte, programs map[[32]byte][]byte,
		computeBudget uint64, blockTime int64, opts RuntimeOptions) (*ExecutionResult, error)
}
