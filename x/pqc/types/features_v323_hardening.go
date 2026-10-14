package types

// v3.2.3 hardening of modules that are disabled, circuit-disabled or quiet on
// every network today (fix plan item C15). Each switch keeps the historical
// behaviour below the plan height, so history replays byte-identically.
const (
	// x/abstractaccount: the EVM authenticator signs the v2 digest, which also
	// binds the gas limit (QSR-2026-0474); registration refuses 65-byte
	// secp256k1 keys that could never verify (QSR-2026-0403) and unknown
	// permission strings, and the three account messages need a valid bech32
	// account (QSR-2026-0436 a, b); daily-spend buckets use length-prefixed
	// keys (QSR-2026-0433).
	FeatureAAHardening = "abstractaccount-hardening"
	// x/rlconsensus: governance messages are range-checked on the raw wire
	// value (agent mode, QSR-2026-0434) and reward weights must each lie in
	// [0, 1] (QSR-2026-0435).
	FeatureRLParamBounds = "rlconsensus-param-bounds"
	// x/amm: EndBlock refreshes at most a bounded number of pools per block,
	// round-robin (QSR-2026-0418).
	FeatureAMMEndBlockBound = "amm-endblock-bound"
	// x/burn: per-burn records older than the retention window are pruned,
	// a bounded number per burn (QSR-2026-0424).
	FeatureBurnRecordPruning = "burn-record-pruning"
)

// V323HardeningFeatures are this area's v3.2.3 features; the release list
// concatenates them.
var V323HardeningFeatures = []string{
	FeatureAAHardening,
	FeatureRLParamBounds,
	FeatureAMMEndBlockBound,
	FeatureBurnRecordPruning,
}
