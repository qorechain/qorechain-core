package types

const (
	// FeatureCrossVMBridgeCodePin binds the EVM bridge allowlist to code, not
	// only to a contract address. Governance may pin, per allowlist entry, the
	// CosmWasm code ID the contract must run (crossvm MsgUpdateParams
	// bridge_target_pins). From this switch:
	//   - the bridge refuses a call to a pinned entry whose contract no longer
	//     runs the pinned code (its admin migrated it);
	//   - the bridge refuses a call to an unpinned entry whose contract has an
	//     admin other than governance (or no admin at all is fine), since that
	//     admin could swap the code behind the entry;
	//   - a parameter update refuses an unpinned entry whose contract has an
	//     admin other than governance, a pin whose code ID is not the
	//     contract's current code, and an entry naming no contract.
	// The release handler pins every existing entry to the code its contract
	// runs at the upgrade height, so every target reachable before stays
	// reachable after. Below the switch a pin cannot be set and nothing is
	// checked beyond the address match.
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
)

// V328WasmAuthzFeatures are the v3.2.8 switches for the CosmWasm lane
// authorization hardening.
var V328WasmAuthzFeatures = []string{
	FeatureCrossVMBridgeCodePin,
	FeatureEVMDispatchTopLevelOnly,
	FeatureCrossVMQueueSenderCap,
}
