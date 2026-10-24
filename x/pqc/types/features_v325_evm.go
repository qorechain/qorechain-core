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
)

// V325EVMFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325EVMFeatures = []string{FeatureEVMAIAnomalyCheckV2}
