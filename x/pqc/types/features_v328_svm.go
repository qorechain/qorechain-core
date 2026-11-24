package types

// v3.2.8 x/svm hardening from the 2026-11-07 internal SVM audit. None of these
// opens the SVM lane: it stays closed by the compile-time switch in the keeper
// and by params.Enabled, and opening it is a separate governance decision.
// They are what has to be true before that decision. Below the v3.2.8 plan
// height all of them are off, so history replays byte-identically: the keeper
// keeps calling the pre-v3.2.8 runtime entries with the pre-v3.2.8 arguments,
// and the runtime switches below are passed as flags to the v3 entry only when
// at least one of them is on.
const (
	// FeatureSVMResultFullFields makes the runtime return owner, executable and
	// rent_epoch for every account, and makes the keeper check all of them for
	// every returned account, under a native program as well as a user one: a
	// read-only account must come back unchanged, the executable flag never
	// changes, an executable account can only be credited, and owner, data and
	// lamport changes follow the ownership rules (native programs only within
	// what each built-in legitimately does). Accounts that did not change are
	// not rewritten. Before it the runtime returned only address, lamports and
	// data, every returned account was persisted with owner 0 and executable
	// false, native programs were exempt from every rule, and any returned
	// read-only account failed the instruction.
	FeatureSVMResultFullFields = "svm-result-full-fields"
	// FeatureSVMDebitNeedsSigner makes any lamport decrease, owner change or
	// data change of a system-owned account under a user program require that
	// account to be a verified signer. Before it a system-owned, empty account
	// that was not a canonical wallet (a real Solana pubkey, a zero-space PDA
	// vault) counted as "claimable": any program could move its lamports to a
	// wallet, and settlement paid the wallet out of module custody.
	FeatureSVMDebitNeedsSigner = "svm-debit-needs-signer"
	// FeatureSVMCPIDepthBounded carries the cross-program invocation depth into
	// a user-program callee, so the depth limit of 4 bounds the whole call tree
	// and sol_get_stack_height reports the real depth. Before it every callee
	// restarted at depth 0, recursion was bounded only by the compute budget,
	// and a callee always saw a stack height of 1.
	FeatureSVMCPIDepthBounded = "svm-cpi-depth-bounded"
	// FeatureSVMTokenOwnerChecks makes the token program refuse a mint or token
	// account that the token program does not own, on every instruction and
	// when reached through ATA or a cross-program call. Before it any account
	// with a token-account layout was accepted, whoever owned it.
	FeatureSVMTokenOwnerChecks = "svm-token-owner-checks"
	// FeatureSVMAuthDigestSigned makes a relayed (foreign-scheme) authorisation
	// of MsgExecuteProgram consumed under the digest that was signed and
	// checked. Before it the digest was recomputed after the wallet key had
	// been replaced by the canonical account, so the consumed digest never
	// matched the checked one and the same authorisation could run again until
	// its blockhash expired.
	FeatureSVMAuthDigestSigned = "svm-auth-digest-signed"
	// FeatureSVMMemcpyNoOverlap makes sol_memcpy_ refuse overlapping ranges.
	// Before it they were copied with a non-overlapping copy, whose result for
	// overlapping ranges is undefined and can differ between hosts.
	FeatureSVMMemcpyNoOverlap = "svm-memcpy-no-overlap"
	// FeatureSVMPDASeedLimits makes the PDA syscalls refuse more than 16 seeds
	// or a seed longer than 32 bytes before hashing. Before it seed count and
	// length were unchecked, so one flat-priced call could hash megabytes.
	FeatureSVMPDASeedLimits = "svm-pda-seed-limits"
	// FeatureSVMConsensusSysvars feeds the Clock (slot = block height, epoch)
	// and Rent (the module's rent params) sysvars from the chain. Before it slot
	// and epoch were 0 and rent was a constant in the runtime.
	FeatureSVMConsensusSysvars = "svm-consensus-sysvars"
	// FeatureSVMPQCKeyLaneGated refuses MsgRegisterSVMPQCKey while the SVM lane
	// is disabled, in the ante handler and in the message handler. Before it the
	// message wrote state with the lane closed.
	FeatureSVMPQCKeyLaneGated = "svm-pqc-key-lane-gated"
	// FeatureSVMNonzeroReturnFails makes a non-zero value returned by a
	// program's entrypoint a program error, as on Solana: the instruction
	// fails with the value reported (e.g. "custom program error: 0x1"), every
	// state change of the message is discarded and the fee is charged as for
	// any failed transaction. A callee returning non-zero fails its caller's
	// invocation the same way. It also makes an error raised inside a syscall,
	// a failed cross-program invocation included, keep its own kind and code
	// (the callee's error, not a generic one). Before it any return value was
	// success, and every error inside a callee surfaced as "syscall error
	// (code -4)".
	FeatureSVMNonzeroReturnFails = "svm-nonzero-return-fails"
)

// V328SVMFeatures are the v3.2.8 switches for SVM hardening and enablement.
var V328SVMFeatures = []string{
	FeatureSVMResultFullFields,
	FeatureSVMDebitNeedsSigner,
	FeatureSVMCPIDepthBounded,
	FeatureSVMTokenOwnerChecks,
	FeatureSVMAuthDigestSigned,
	FeatureSVMMemcpyNoOverlap,
	FeatureSVMPDASeedLimits,
	FeatureSVMConsensusSysvars,
	FeatureSVMPQCKeyLaneGated,
	FeatureSVMNonzeroReturnFails,
}
