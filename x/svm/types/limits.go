package types

// Per-instruction bounds and gas schedule of the SVM lane (v3.2.5,
// svm-execution-bounds and svm-compute-gas). Before them an instruction could
// name any number of accounts, carry any amount of data and make the keeper load
// up to 10 MiB of bytecode for every executable account it named, and none of
// the execution was charged to the transaction gas meter.
const (
	// MaxAccountsPerInstruction is the number of account metas one instruction
	// may name. 32 is above what any Solana program needs in a single
	// instruction without address lookup tables, which QoreChain does not have.
	MaxAccountsPerInstruction = 32

	// MaxInstructionDataLen is Solana's packet-derived limit on instruction
	// data (1232 bytes, the IPv6 MTU less headers).
	MaxInstructionDataLen = 1232

	// MaxAccountDataPerInstruction bounds the account data plus program
	// bytecode one instruction makes the keeper load, serialize, hand to the
	// runtime and copy back: 32 MiB.
	MaxAccountDataPerInstruction = 32 << 20
)

// Gas schedule. Calibrated on 2026-10-25 against the EVM lane, which already
// charges its gas to the same meter: on the same machine the go-ethereum
// interpreter ran an ADD/SUB/JUMPI loop at 1.15 ns per gas, and the SVM
// interpreter ran an ALU loop at 1.38 ns per compute unit (1.2M CU in 1.65 ms).
// The other costs are the work the keeper does per account and per byte that
// the store gas does not already cover: decoding an account record from JSON
// costs 3.4 ns per byte, serializing it into the runtime's input format 0.08 ns
// per byte, and the fixed per-account work (decode, encode, two base58
// conversions) 5.5 us, of which the store's flat read charge covers about a
// third.
const (
	// GasPerComputeUnit: 1.38 / 1.15 = 1.2, rounded up.
	GasPerComputeUnit = 2

	// GasPerSerializedByte: (3.4 + 0.08 + runtime copy and parse) ns per byte
	// over 1.15 ns per gas is about 3.3, rounded up. Charged for every byte of
	// account data and program bytecode handed to the runtime, for the
	// instruction data, and for every byte of account data copied back.
	GasPerSerializedByte = 4

	// GasPerAccountMeta: 5.5 us is about 4,800 gas, less the ~1,700 the store
	// already charges for reading a small account record.
	GasPerAccountMeta = 3000
)
