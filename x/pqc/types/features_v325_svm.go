package types

// Released in v3.2.5: the SVM hardening found by the 2026-10-25 audit. The SVM
// lane is still closed by the compile-time switch in the keeper, so none of
// these is reachable on a live network today; they are what has to be true
// before the lane can be reopened. Below the v3.2.5 plan height all four are
// off, so history replays byte-identically.
const (
	// x/svm: deploying a program debits the deployer the rent-exempt lamports
	// of the program and data accounts it creates, moving them into the svm
	// module's custody. Before it the data account was created holding
	// lamports nobody paid for, and the program that owns it could hand them to
	// a wallet, which the module then paid out as real QOR from custody that
	// belongs to other accounts. Executable accounts can no longer be claimed
	// or have their executable flag changed by a user program.
	FeatureSVMDeployRentFunded = "svm-deploy-rent-funded"
	// x/svm: an instruction may name at most 32 accounts, carry at most 1232
	// bytes of instruction data, and load at most 32 MiB of account data and
	// program bytecode in total. Checked in the ante handler (CheckTx and
	// DeliverTx alike) and again in the keeper, which also covers the cross-VM
	// route. MsgExecuteProgram is refused in the ante handler while the lane is
	// disabled, so it no longer pays the SVM fee for a call that cannot run.
	FeatureSVMExecutionBounds = "svm-execution-bounds"
	// x/svm: execution is charged to the transaction gas meter: a fixed amount
	// per account, per serialized byte and per compute unit, with the compute
	// budget capped by the gas the transaction has left. The runtime charges
	// syscalls (hashing, memory operations, logging, PDA derivation, CPI) by
	// the bytes they touch. The SVM fee decorator does the same store work in
	// simulate as in DeliverTx, so a gas estimate matches the charge.
	FeatureSVMComputeGas = "svm-compute-gas"
	// x/svm: the lamport balance of a non-canonical 32-byte address (one whose
	// tail is not zero) is read from the SVM store only. Before it the read
	// truncated the address to 20 bytes and reported an unrelated native
	// account's balance.
	FeatureSVMNativeReadCanonical = "svm-native-read-canonical"
)

// V325SVMFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325SVMFeatures = []string{
	FeatureSVMDeployRentFunded,
	FeatureSVMExecutionBounds,
	FeatureSVMComputeGas,
	FeatureSVMNativeReadCanonical,
}
