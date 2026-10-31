package types

// v3.2.5 EVM area switches. Below the v3.2.5 plan height every one of them is
// off, so history replays byte-identically.
const (
	// FeatureEVMAIAnomalyCheckV2 makes the aiAnomalyCheck precompile (0x0B02)
	// answer with x/ai screening v2, the deterministic rules the ante handler
	// applies, over the account's stored history and rate window, read-only
	// and off the gas meter. Below it the precompile asks the legacy float
	// engine with no history. The precompile is inactive on every network
	// until governance adds it to active_static_precompiles.
	FeatureEVMAIAnomalyCheckV2 = "evm-ai-anomaly-check-v2"

	// FeatureEVMPrecompileWindowScope makes the value-moving standard
	// precompiles (staking, distribution, ics20, gov; see
	// EVMValueMovingPrecompiles) part of the EVM authorisation window's scope.
	// They move or commit the caller's funds with a transaction value of
	// zero, so the window's value bound cannot see what they do, and an
	// unbonding, a redelegation or a new withdraw address moves no balance at
	// all. From the switch a transaction method of one of them runs for an
	// EVM-lane account (the transaction's sender, or an EIP-7702 delegated
	// account whose code calls it) only when that account's window lists the
	// precompile in allowed_targets, and an EVM transaction addressed to one
	// of them is refused in the ante handler unless the window lists it. The
	// standard precompiles are inactive on every network until governance
	// adds them to active_static_precompiles.
	FeatureEVMPrecompileWindowScope = "evm-precompile-window-scope"

	// FeatureEVMPrecompileCircuit makes every transaction method of a standard
	// precompile honour the circuit-breaker entry of the Cosmos message it
	// executes (staking delegate honours MsgDelegate, ics20 transfer honours
	// MsgTransfer, and so on). The precompiles call the module message
	// servers directly, so without the switch a tripped message stays
	// reachable from the EVM.
	FeatureEVMPrecompileCircuit = "evm-precompile-circuit"
)

// V325EVMFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325EVMFeatures = []string{
	FeatureEVMAIAnomalyCheckV2,
	FeatureEVMPrecompileWindowScope,
	FeatureEVMPrecompileCircuit,
}
