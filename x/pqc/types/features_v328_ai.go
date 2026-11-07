package types

const (
	// FeatureEVMAIAnomalyCheckReader lets the aiAnomalyCheck precompile
	// (0x0B02) read x/ai screening state through the keeper the app actually
	// gives it. With evm-ai-anomaly-check-v2 on, the precompile asks its keeper
	// for three read methods (fresh config, rate window count, history ring);
	// the app's x/ai keeper wrapper did not forward them, so the precompile
	// always took its fail-closed branch and answered (10000, true), flagged,
	// for every account and amount. From this switch the wrapper's forwarders
	// are used and the answer is the screening v2 verdict the ante handler
	// gives the same transfer. Below it the precompile answers (10000, true)
	// exactly as before. 0x0B02 is not an active precompile on either network,
	// so no contract could call it either way.
	FeatureEVMAIAnomalyCheckReader = "evm-ai-anomaly-check-reader"
)

// V328AIFeatures are the v3.2.8 switches for AI precompile wiring.
var V328AIFeatures = []string{
	FeatureEVMAIAnomalyCheckReader,
}
