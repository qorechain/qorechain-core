package types

const (
	// FeatureCrossVMBelowBridgeOwnSender refuses a MsgCrossVMCall handled
	// below the EVM bridge precompile (a CosmWasm contract the bridge called,
	// or anything that contract dispatched) unless its sender is a CosmWasm
	// contract acting as itself. The EVM-lane account that called the bridge
	// is authorised only by its EVM window, whose scope covers the bridge
	// call; a message naming that account (or any other account) dispatched
	// below it, through an authz grant held by the contract, would otherwise
	// run as that account with no window, sync into the SVM or queued into
	// the EVM. An authz MsgExec below the bridge that carries a
	// MsgCrossVMCall for another account is refused for the same reason. A
	// contract calling out as itself, sync to the SVM or async to the EVM,
	// is unaffected.
	FeatureCrossVMBelowBridgeOwnSender = "crossvm-below-bridge-own-sender"

	// FeatureCrossVMOnBehalfEVMWindow applies the EVM authorisation window to
	// a cross-VM call into the EVM that an account did not send itself. A
	// MsgCrossVMCall into the EVM runs as the sender's EVM identity; when it
	// is not one of the transaction's own top-level messages and its sender
	// is not a CosmWasm contract, a group policy or the governance account
	// acting as itself, it was sent on the account's behalf (an authz grant
	// exercised by a person, a contract or a group policy), and from this
	// switch it runs exactly as the account's delegated code would in
	// someone else's EVM transaction:
	//   - the account must hold a usable post-quantum key and a live window,
	//     whose scope admits the target (allowed_targets, native_only, the
	//     value-moving precompiles), with a transaction left; the call uses
	//     one window transaction when it succeeds;
	//   - the EVM execution settles the account as an EVM-lane sender, so
	//     native value it loses and native allowances it grants are bounded
	//     by what remains of the window's value and charged to it, and the
	//     standard precompiles treat it as an EVM-lane account (authority
	//     methods refused, value-moving precompiles only when listed);
	//   - without a window the call is refused, at submission for an async
	//     call and again when the queue runs it.
	// An authz grant from a CosmWasm contract or a group policy (which can
	// hold no window) cannot carry a MsgCrossVMCall into the EVM. A
	// MsgCrossVMCall the account signs itself at top level with its
	// post-quantum key, and a contract or group policy calling as itself,
	// execute exactly as before.
	FeatureCrossVMOnBehalfEVMWindow = "crossvm-on-behalf-evm-window"

	// FeatureCrossVMTruncatedCallerNoValue refuses a cross-VM call into the
	// EVM from a 32-byte sender (a CosmWasm contract, a group policy) whose
	// execution leaves native QOR credited to the 20-byte account its EVM
	// identity is derived from (the last 20 bytes of its address). No
	// cross-VM path carries native value back out of that account, so the
	// coins would be stranded there. The call fails (and is charged its gas)
	// instead; calls that credit it nothing, ERC-20 transfers included, are
	// unaffected. docs/CROSSVM.md describes this for dApp authors.
	FeatureCrossVMTruncatedCallerNoValue = "crossvm-truncated-caller-no-value"

	// FeatureCrossVMQueueHeightIndex bounds the end-of-block queue walk by
	// what one block can do instead of by the queue's length. Each queued
	// call is indexed by creation height and message ID (the queue's FIFO
	// order); a block times out at most max_queue_timeouts_per_block calls
	// (crossvm params, default DefaultMaxQueueTimeoutsPerBlock), oldest
	// first, reading only the index keys below the timeout height, and then
	// visits at most MaxQueueVisitsPerBlock calls that have not timed out,
	// in FIFO order, executing at most MaxAsyncCallsPerBlock of them. Calls
	// queued before the switch are indexed once, at the first block that
	// walks the queue from it. Which calls run, in which order, is unchanged
	// while the queue holds no more than one block's worth of timeouts and
	// visits; a backlog is worked through over the following blocks.
	FeatureCrossVMQueueHeightIndex = "crossvm-queue-height-index"

	// FeatureERC20ConvertOnBehalfEVMWindow applies the EVM authorisation
	// window to a MsgConvertERC20 executed under an authz grant (by a
	// person, a contract or a group policy, directly or as a nested
	// self-exec). The conversion runs the token contract with the sender as
	// msg.sender, so it is an EVM action of the sender: from this switch the
	// sender must hold a usable post-quantum key and a live window whose scope
	// admits a call to the token contract, with a transaction left, and the
	// conversion uses one window transaction. A conversion the account sends
	// itself is unchanged.
	FeatureERC20ConvertOnBehalfEVMWindow = "erc20-convert-on-behalf-evm-window"

	// FeatureCrossVMQueueSenderRoundRobin makes the end-of-block queue take
	// its senders in turn. From the height index (crossvm-queue-height-index)
	// the walk reads up to MaxQueueScanPerBlock index keys that have not
	// timed out, each carrying its call's sender, and lists them one call per
	// sender per round (senders in the order of their oldest call, each
	// sender's calls oldest first), up to MaxQueueVisitsPerBlock calls. One
	// sender with many calls queued ahead of others no longer keeps their
	// calls from running until they time out; a queue with one sender runs
	// exactly as before. Only calls the block admits count against
	// MaxQueueVisitsPerBlock: a call that waits (its price or its lane's
	// budget is not met) is skipped, without being decoded when its index
	// value shows it waiting, so waiting calls cannot take the visits of
	// calls that can run.
	FeatureCrossVMQueueSenderRoundRobin = "crossvm-queue-sender-round-robin"
)

// V328XVMAuthzFeatures are the v3.2.8 switches for the cross-VM
// authorisation review.
var V328XVMAuthzFeatures = []string{
	FeatureCrossVMBelowBridgeOwnSender,
	FeatureCrossVMOnBehalfEVMWindow,
	FeatureCrossVMTruncatedCallerNoValue,
	FeatureCrossVMQueueHeightIndex,
	FeatureERC20ConvertOnBehalfEVMWindow,
	FeatureCrossVMQueueSenderRoundRobin,
}
