package types

// v3.2.7 x/crossvm switches. Below the v3.2.7 plan height both are off, so
// history replays byte-identically.
const (
	// FeatureCrossVMPanicContained makes a panic inside a cross-VM call the
	// call's failure instead of the block's. In the end-of-block queue every
	// call runs in its own cache branch on the gas budget it prepaid; a panic
	// of any kind (running out of that budget included) drops the branch,
	// records the message as failed with a fixed error text and emits a
	// crossvm_response event carrying that error, and the queue moves on to
	// the next call. On the synchronous path (MsgCrossVMCall without async,
	// and the 0x0901 bridge) a panic that is not out-of-gas becomes an
	// ErrExecutionFailed error, so the transaction fails with a typed error;
	// out-of-gas still propagates as before, so the transaction's gas
	// accounting is unchanged. Before it the queue re-raised every panic that
	// was not out-of-gas, so one queued call could stop the chain: a 32-byte
	// sender (a group policy account) queuing a call into SVM made the
	// crossvm EndBlock panic on every node (svm-address-audit F1).
	FeatureCrossVMPanicContained = "crossvm-panic-contained"

	// FeatureCrossVMAddressShapes refuses, when a cross-VM call is submitted
	// (queued or synchronous, from a transaction or from the 0x0901 bridge),
	// a sender or a target the target lane cannot represent: a sender that is
	// neither 20 nor 32 bytes; for SVM a sender that is not a 20-byte account
	// (only those have an SVM identity) or a target that is not a canonical
	// Base58 32-byte address; for EVM a target that is not 0x followed by 40
	// hex digits, or that is a precompile address (at most 0xffff, the cross-VM
	// bridge included); for CosmWasm a target that is not a 32-byte bech32
	// account address. Before it such a call was accepted and failed, or
	// panicked, only when it executed, which for a queued call is in EndBlock.
	FeatureCrossVMAddressShapes = "crossvm-address-shapes"
)

// V327CrossVMFeatures are the v3.2.7 switches that make the cross-VM call safe and usable.
var V327CrossVMFeatures = []string{
	FeatureCrossVMPanicContained,
	FeatureCrossVMAddressShapes,
}
