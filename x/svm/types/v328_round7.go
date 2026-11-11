package types

import (
	"encoding/binary"
	"errors"
)

// ErrExecutionThreadUnavailable is returned by the runtime bridge when the
// fixed-stack execution thread could not be created on this node even after
// the runtime's bounded retries. It is a plain (unregistered) error so a
// failed transaction keeps the same ABCI code it had before the error was
// named. With svm-execution-thread-node-fault, during block execution it
// becomes a NodeFault panic instead of a transaction result.
var ErrExecutionThreadUnavailable = errors.New("svm execution thread unavailable on this node")

// NodeFault is the panic value of a node-local failure that must not become a
// transaction result (svm-execution-thread-node-fault): the result would
// differ from every other node's. NodeFaultRecoveryHandler re-raises it out
// of the transaction, so block execution stops and the node halts before
// committing anything; on restart the block is executed again.
type NodeFault struct {
	Reason string
}

func (f NodeFault) Error() string {
	return "svm node fault (this node cannot execute the block; nothing was committed): " + f.Reason
}

// NodeFaultRecoveryHandler is a baseapp RecoveryHandler. Every other panic is
// left to the next handler (it returns nil for them).
func NodeFaultRecoveryHandler(recoveryObj interface{}) error {
	if f, ok := recoveryObj.(NodeFault); ok {
		panic(f)
	}
	return nil
}

// SVMPQCPublicKeyLen is the length of a registered SVM PQC public key
// (ML-DSA-87 / Dilithium-5), as MsgRegisterSVMPQCKey requires.
const SVMPQCPublicKeyLen = 2592

// MaxNativeAllocation is the largest account a native System instruction can
// allocate in one go (Solana's 10 MiB MAX_PERMITTED_DATA_LENGTH).
const MaxNativeAllocation = 10 << 20

// NativeAllocation is the account space a top-level System instruction asks
// for (CreateAccount, Allocate and their WithSeed forms), at most
// MaxNativeAllocation; zero for anything else. It is exactly the amount the
// runtime bridge adds to the result buffer for the instruction (and, with
// svm-native-allocation-charged, what the keeper charges for before running
// it), malformed data included.
func NativeAllocation(programID [32]byte, ix []byte) int {
	if programID != SystemProgramAddress || len(ix) < 4 {
		return 0
	}
	d := ix[4:]
	u64 := func(off int) (uint64, bool) {
		if off < 0 || len(d) < off+8 {
			return 0, false
		}
		return binary.LittleEndian.Uint64(d[off : off+8]), true
	}
	afterSeed := func() int { // base(32) | seed_len(8) | seed
		n, ok := u64(32)
		if !ok || n > 32 {
			return -1
		}
		return 40 + int(n)
	}
	var space uint64
	var ok bool
	switch binary.LittleEndian.Uint32(ix[:4]) {
	case 0: // lamports, space, owner
		space, ok = u64(8)
	case 8: // space
		space, ok = u64(0)
	case 3: // base, seed, lamports, space, owner
		space, ok = u64(afterSeed() + 8)
	case 9: // base, seed, space, owner
		space, ok = u64(afterSeed())
	}
	if !ok {
		return 0
	}
	if space > MaxNativeAllocation {
		space = MaxNativeAllocation
	}
	return int(space)
}

// nativeProgramIDs are addresses no instruction can ever spend from: the
// built-in programs, the loader, the QoreChain built-ins, the sysvars and the
// other well-known Solana program ids (svm-no-credit-native-program).
var nativeProgramIDs = func() map[[32]byte]bool {
	ids := [][32]byte{
		SystemProgramAddress, SPLTokenAddress, ATAAddress, MemoAddress,
		QorPQCAddress, QorAIAddress, BPFLoaderAddress,
	}
	for _, s := range []string{
		"metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s", // token metadata
		"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb", // token-2022
		"BPFLoader1111111111111111111111111111111111",
		"BPFLoader2111111111111111111111111111111111",
		"NativeLoader1111111111111111111111111111111",
		"ComputeBudget111111111111111111111111111111",
		"Config1111111111111111111111111111111111111",
		"Stake11111111111111111111111111111111111111",
		"Vote111111111111111111111111111111111111111",
		"AddressLookupTab1e1111111111111111111111111",
		"Ed25519SigVerify111111111111111111111111111",
		"KeccakSecp256k11111111111111111111111111111",
		"Sysvar1111111111111111111111111111111111111",
		"SysvarC1ock11111111111111111111111111111111",
		"SysvarRent111111111111111111111111111111111",
		"SysvarEpochSchedu1e111111111111111111111111",
		"SysvarFees111111111111111111111111111111111",
		"SysvarRecentB1ockHashes11111111111111111111",
		"SysvarS1otHashes111111111111111111111111111",
		"SysvarS1otHistory11111111111111111111111111",
		"SysvarStakeHistory1111111111111111111111111",
		"Sysvar1nstructions1111111111111111111111111",
		"SysvarEpochRewards1111111111111111111111111",
		"SysvarLastRestartS1ot1111111111111111111111",
	} {
		ids = append(ids, mustBase58Decode(s))
	}
	m := make(map[[32]byte]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}()

// IsNativeProgramID reports whether addr is a native program, loader or
// sysvar id (svm-no-credit-native-program).
func IsNativeProgramID(addr [32]byte) bool { return nativeProgramIDs[addr] }
