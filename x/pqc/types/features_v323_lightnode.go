package types

// v3.2.3 light node reward bound (decision of 2026-08-24, measured drift
// 2026-10-14). Below the v3.2.3 plan height it is off, so history replays
// byte-identically.
const (
	// The per-node reward bound is 2,000 QOR per 30 days prorated over block
	// time, instead of a per-block constant that only matched the decision at
	// the cadence it was derived at.
	FeatureLightnodeMonthlyCap = "lightnode-monthly-cap"
)

// V323LightnodeFeatures is this area's share of the v3.2.3 release set.
var V323LightnodeFeatures = []string{FeatureLightnodeMonthlyCap}
