package types

// Released in v3.2.3: the cross-VM queue and the EVM precompile set. Below the
// v3.2.3 plan height both switches are off, so history replays byte-identically.
const (
	// x/crossvm (QSR-2026-0422, -0426, -0379): a queued call that fails leaves
	// no side effects behind, only its Failed record; the queue is not executed
	// while the module is disabled or MsgCrossVMCall is circuit-tripped; and a
	// call whose derived source lane equals its target lane is refused.
	FeatureCrossVMQueueAtomicity = "crossvm-queue-atomicity"
	// EVM precompiles (QSR-2026-0394, -0398, -0395, -0417): the CrossVM bridge
	// refuses STATICCALL, meters its nested call against the calling frame's
	// gas, reaches only allowlisted CosmWasm targets and honours the circuit
	// breaker; the AI anomaly precompile analyses without writing.
	FeatureEVMPrecompileHardening = "evm-precompile-hardening"
)

// V323CrossvmPrecompileFeatures is this area's share of the v3.2.3 release
// switches.
var V323CrossvmPrecompileFeatures = []string{
	FeatureCrossVMQueueAtomicity,
	FeatureEVMPrecompileHardening,
}
