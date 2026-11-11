package types

import "sync"

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
