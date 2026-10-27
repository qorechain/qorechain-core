package types

const (
	// FeatureRLParamsPrecompileLive makes the rlConsensusParams precompile
	// (0x0C01) return the parameters x/rlconsensus holds: the applied block
	// time and gas price floor, the module's validator set size and the
	// agent's current epoch. Before it the precompile answered the static
	// genesis defaults (5000, 100, 100, 0) whatever the module held. Read-only;
	// the flat 1,500 gas stays and the store reads are charged on top at the
	// store's read cost.
	FeatureRLParamsPrecompileLive = "rl-params-precompile-live"
)

// V326Features is the v3.2.6 release set. Every consensus change of the release
// is guarded by one of these switches, so below the v3.2.6 plan height history
// replays byte-identically.
var V326Features = []string{
	FeatureRLParamsPrecompileLive,
	FeatureRLAgentObservable,
	FeatureRLCircuitBreakerPerBlock,
	FeatureRLHistoryRetention,
}
