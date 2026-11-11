package types

const (
	// FeatureCrossVMBridgeCodePin binds the EVM bridge allowlist to code and
	// state, not only to a contract address. Governance may pin, per
	// allowlist entry, the CosmWasm code ID the contract must run and the
	// length of its code history at the time of the pin (crossvm
	// MsgUpdateParams bridge_target_pins). Every migration lengthens the
	// history, so a contract migrated to other code (which may rewrite its
	// storage) and back no longer matches. From this switch:
	//   - the bridge refuses a call to a pinned entry whose contract runs
	//     other code or has been migrated since the pin;
	//   - the bridge refuses a call to an unpinned entry whose contract does
	//     not exist or has an admin other than governance (no admin is fine),
	//     since that admin could swap the code behind the entry;
	//   - a parameter update refuses an unpinned entry whose contract has an
	//     admin other than governance, a pin that does not match the
	//     contract's current code and history, and an entry naming no
	//     contract.
	// The release handler pins every existing entry to the code and history
	// its contract has at the upgrade height, so every target reachable before
	// stays reachable after, and removes (with an event) entries that name no
	// contract. Below the switch a pin cannot be set and nothing is checked
	// beyond the address match.
	FeatureCrossVMBridgeCodePin = "crossvm-bridge-code-pin"

	// FeatureEVMDispatchTopLevelOnly makes the MsgEthereumTx dispatch guard
	// count the transaction's own MsgEthereumTx messages instead of trusting a
	// context mark that every nested call inherits. The EVM-lane ante handler
	// records how many MsgEthereumTx the transaction carries at top level; each
	// execution of the handler takes one, so a MsgEthereumTx dispatched from
	// below one of them (a contract reached through the cross-VM bridge, for
	// example) finds none left and is refused. Every top-level EVM-lane
	// transaction executes exactly as before.
	FeatureEVMDispatchTopLevelOnly = "evm-dispatch-top-level-only"

	// FeatureCrossVMQueueSenderCap bounds how many async cross-VM calls one
	// sender may have pending at a time (crossvm params max_pending_per_sender,
	// default DefaultMaxPendingPerSender), so one sender cannot fill the shared
	// queue and lock everyone else out of the async path. Each pending call is
	// indexed by sender when it is queued and unindexed when it leaves the
	// queue, so the check reads at most the cap's worth of keys. Callers under
	// the cap, and synchronous calls, are unaffected.
	FeatureCrossVMQueueSenderCap = "crossvm-queue-sender-cap"

	// FeatureCrossVMFailedCallCharged charges every cross-VM call for the
	// work it did when it fails, as it is charged when it succeeds. Before it,
	// an EVM call made through x/crossvm that reverted or ran out of gas
	// charged nothing for the EVM execution (only the store reads), and a
	// CosmWasm call made through the 0x0901 bridge that failed charged its
	// caller's EVM frame nothing for the contract's execution; a caller that
	// caught the failure (a contract reply, an x/group proposal with
	// Exec=TRY, an EVM try/catch) could repeat it on the same remaining gas.
	// From the switch the EVM gas a failed sub-call used is consumed from the
	// caller's meter before the failure is reported, and a failed bridge call
	// consumes the gas the contract used from the calling EVM frame, so the
	// work one transaction can cause is bounded by its gas limit. Calls that
	// succeed cost exactly what they did.
	FeatureCrossVMFailedCallCharged = "crossvm-failed-call-charged"

	// FeatureCrossVMErrorTextPlain keeps build-dependent text out of
	// consensus data. Formatting an SDK error with %w or %v appends a
	// " [<path>.go:<line>]" annotation whose path depends on how the node was
	// built, and the 0x0901 bridge put that text into the revert data it
	// returns to the EVM caller (part of the transaction result, readable and
	// storable by the caller). From the switch the bridge builds its revert
	// reasons from plain error text, and the error recorded on a failed queued
	// cross-VM call is stripped of such annotations too.
	FeatureCrossVMErrorTextPlain = "crossvm-error-text-plain"

	// FeatureCrossVMQueueLazyDecode bounds the end-of-block queue walk. Before
	// it every block loaded and decoded every pending call, payload included,
	// though at most MaxAsyncCallsPerBlock of them run. From the switch a call
	// queued records its creation height in a small per-call key, and the walk
	// reads those keys to find the timed-out calls, decoding only the calls it
	// times out or executes in that block. Which calls run or time out, and in
	// which order, is unchanged.
	FeatureCrossVMQueueLazyDecode = "crossvm-queue-lazy-decode"
)

// V328WasmAuthzFeatures are the v3.2.8 switches for the CosmWasm lane
// authorization hardening.
var V328WasmAuthzFeatures = []string{
	FeatureCrossVMBridgeCodePin,
	FeatureEVMDispatchTopLevelOnly,
	FeatureCrossVMQueueSenderCap,
	FeatureCrossVMFailedCallCharged,
	FeatureCrossVMErrorTextPlain,
	FeatureCrossVMQueueLazyDecode,
}
