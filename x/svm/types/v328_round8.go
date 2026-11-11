package types

import (
	"errors"
	"fmt"
	"sync"
)

// FullProgramValidator is a runtime that can run, on a program being
// deployed, the full loader and verifier an execution runs
// (svm-deploy-fully-verified).
type FullProgramValidator interface {
	ValidateProgramFull(bytecode []byte) error
}

// The node-fault latch. A NodeFault is a panic, and some callers between the
// SVM and the transaction recover every panic (x/gov runs proposal messages
// under such a recover, and so may others). The latch keeps the fault even
// when the panic is swallowed: the app's FinalizeBlock reads it after the
// block and fails the block, so the node halts before committing instead of
// committing a result other nodes did not compute.
var (
	nodeFaultMu sync.Mutex
	nodeFault   *NodeFault
)

// RaiseNodeFault records f in the latch and panics with it.
func RaiseNodeFault(f NodeFault) {
	nodeFaultMu.Lock()
	if nodeFault == nil {
		nodeFault = &f
	}
	nodeFaultMu.Unlock()
	panic(f)
}

// PendingNodeFault returns the latched fault, or nil.
func PendingNodeFault() *NodeFault {
	nodeFaultMu.Lock()
	defer nodeFaultMu.Unlock()
	return nodeFault
}

// ClearNodeFault empties the latch (at the start of each block, and in tests).
func ClearNodeFault() {
	nodeFaultMu.Lock()
	nodeFault = nil
	nodeFaultMu.Unlock()
}

// ErrRuntimeFlagsUnsupported is returned by the runtime bridge when the SVM
// library refused a runtime switch it does not know (code -38): the library
// is older than the chain binary. A plain error, like
// ErrExecutionThreadUnavailable; with svm-library-flags-checked it is a node
// fault during block execution.
var ErrRuntimeFlagsUnsupported = errors.New("svm library does not support a runtime switch this binary uses")

// AllRuntimeFlags is every v3 runtime flag this binary can request. The SVM
// library must know all of them; the node refuses to start otherwise.
const AllRuntimeFlags = RuntimeFlagByteMetering | RuntimeFlagFullResult | RuntimeFlagTokenOwnerCheck |
	RuntimeFlagCPIDepthBounded | RuntimeFlagMemcpyNoOverlap | RuntimeFlagPDASeedLimits |
	RuntimeFlagConsensusSysvars | RuntimeFlagNonzeroReturn | RuntimeFlagHostPrivileges |
	RuntimeFlagTokenSPL | RuntimeFlagMetadataNotNat | RuntimeFlagATAIdempotent |
	RuntimeFlagCPICalleeRules | RuntimeFlagATAPrefunded | RuntimeFlagTokenCloseFrozen |
	RuntimeFlagCPICallerRules | RuntimeFlagCPIComposable | RuntimeFlagSystemSeedBase |
	RuntimeFlagPDASigningMeter | RuntimeFlagReentrancyGuard | RuntimeFlagATAPayerSystem |
	RuntimeFlagResultSizeExact | RuntimeFlagSystemSeedCPI | RuntimeFlagNestedAttrib |
	RuntimeFlagResultBounded | RuntimeFlagSeedBases | RuntimeFlagRegistrySharedCharge |
	RuntimeFlagSystemOwnerChecks

// FlagReporter is a runtime that reports the v3 flags it knows.
type FlagReporter interface {
	KnownFlags() uint64
}

// CheckRuntimeFlags returns an error naming the flags `known` lacks.
func CheckRuntimeFlags(known uint64) error {
	if missing := AllRuntimeFlags &^ known; missing != 0 {
		return fmt.Errorf("the SVM library is older than this binary: it does not support runtime flags %#x (it knows %#x, the binary uses %#x); install the libqoresvm built with this release", missing, known, AllRuntimeFlags)
	}
	return nil
}
