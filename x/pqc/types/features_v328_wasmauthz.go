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

	// FeatureWasmFloatOpsRefused extends the CosmWasm code filter to float
	// (f32/f64) operators. The VM meters every operator at one flat price,
	// and float operators cost the validator the most CPU per unit of gas of
	// any (a loop of f64.sqrt ran thousands of times slower per gas than an
	// EVM loop). No release contract, no contract stored on the testnet and
	// none of wasmd's test contracts contains one, so from this switch storing
	// a module with float operators is refused and so is running one. Code
	// without them is unaffected.
	FeatureWasmFloatOpsRefused = "wasm-float-ops-refused"

	// FeatureWasmGasCalibrated prices CosmWasm execution closer to what it
	// costs the validator. From this switch one unit of SDK gas buys 5,000
	// units of VM gas instead of wasmd's 140,000 (app/wasm_gas_register.go),
	// so ordinary contract compute costs about what EVM execution does per
	// unit of gas; storage, events and instance setup are priced as before
	// (typical contract calls cost 11 to 16 percent more gas).
	FeatureWasmGasCalibrated = "wasm-gas-calibrated"

	// FeatureCosmosTxGasCap bounds the gas limit one Cosmos-lane transaction
	// may declare (crossvm params max_cosmos_tx_gas, default
	// DefaultMaxCosmosTxGas, adjustable by governance). With the block gas
	// unbounded, nothing else bounds what one transaction can ask for.
	// Simulation is not affected; EVM-lane transactions keep their own bound.
	FeatureCosmosTxGasCap = "cosmos-tx-gas-cap"

	// FeatureCrossVMResponseCharged charges the data a queued call returns
	// against the call's own prepaid budget, at the store write price, before
	// it is stored. A call whose response does not fit fails like any other
	// call that exceeds its budget (and stores no data). Before, the response
	// was written at the end of the block outside any meter.
	FeatureCrossVMResponseCharged = "crossvm-response-charged"

	// FeatureCrossVMRecordPrune deletes the records of queued calls that have
	// finished (executed, failed or timed out) once they are older than the
	// retention window (crossvm params record_retention_blocks, default
	// DefaultRecordRetentionBlocks), at most MaxRecordPrunePerBlock per block.
	// Before, every record was kept forever.
	FeatureCrossVMRecordPrune = "crossvm-record-prune"

	// FeatureCrossVMAsyncFundsChecked refuses at submission an async call
	// that attaches funds to a lane that cannot carry them, as the
	// synchronous path does at execution, instead of queueing a call that can
	// only fail at the end of the block after its budget was prepaid.
	FeatureCrossVMAsyncFundsChecked = "crossvm-async-funds-checked"

	// FeatureCrossVMReentryCanonical compares call frames by lane and
	// canonical address bytes, so one contract spelled two ways (bech32 upper
	// case, EVM hex case) is one frame for the re-entry guard.
	FeatureCrossVMReentryCanonical = "crossvm-reentry-canonical"

	// FeatureCrossVMQueueFIFO runs the async queue in order of creation
	// height, then message ID, instead of the order of the hashed IDs, so a
	// call queued earlier runs no later than one queued after it.
	FeatureCrossVMQueueFIFO = "crossvm-queue-fifo"

	// FeatureWasmBlockGasBudget bounds the CosmWasm execution of one block.
	// The VM's flat operator price leaves some operator classes (memory.grow,
	// integer division) costing the validator far more CPU per unit of gas
	// than the EVM, so a block full of such contract calls could take tens of
	// seconds to execute. From this switch the VM gas the block's
	// transactions use (execute, instantiate, migrate, sudo, reply and the
	// contract queries they make, whether the transaction succeeds or not) is
	// counted against crossvm params max_block_wasm_gas (default
	// DefaultMaxBlockWasmGas); a VM call is given at most what is left, and
	// once it is spent every further VM call in the block fails ("block wasm
	// gas budget exhausted"). One transaction may use at most its share
	// (max_tx_wasm_share_bps, half by default), so it cannot close the lane
	// for the rest of the block. The end-of-block async queue counts too and
	// leaves a CosmWasm call pending while the budget cannot cover its
	// prepaid gas; a contract call made outside a transaction (a governance
	// proposal) runs on at most max_block_level_wasm_gas and counts, without
	// being refused. The next block starts with the full budget. EVM and
	// native transactions, CheckTx, simulations and queries are not counted.
	FeatureWasmBlockGasBudget = "wasm-block-gas-budget"

	// FeatureWasmCongestionPricing prices CosmWasm execution by how full the
	// block wasm budget has been. A factor kept in state (crossvm,
	// default 1) multiplies the SDK gas one unit of contract execution costs;
	// after every block it rises by 1/8 when the block used more than
	// wasm_congestion_target_bps of its wasm budget and falls by 1/9 when it
	// used less, bounded by wasm_congestion_max_factor (default 1,000). The
	// budget itself is unchanged (it bounds validator CPU); keeping it full
	// becomes exponentially expensive, and ordinary use, which leaves most of
	// the budget unused, stays at the base price.
	FeatureWasmCongestionPricing = "wasm-congestion-pricing"
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
	FeatureWasmFloatOpsRefused,
	FeatureWasmGasCalibrated,
	FeatureCosmosTxGasCap,
	FeatureCrossVMResponseCharged,
	FeatureCrossVMRecordPrune,
	FeatureCrossVMAsyncFundsChecked,
	FeatureCrossVMReentryCanonical,
	FeatureCrossVMQueueFIFO,
	FeatureWasmBlockGasBudget,
	FeatureWasmCongestionPricing,
}
