package types

// v3.2.4 RDK area. Below the v3.2.4 plan height it is off, so history replays
// byte-identically.
const (
	// FeatureRDKChallengeWindowTime measures an optimistic batch's challenge
	// window in seconds of block time, as ChallengeWindowSec says, instead of
	// ChallengeWindowSec/6 blocks. A batch submitted from the switch records
	// its submission block time; finalization and the challenge cut-off for it
	// compare the current block time with submitted_time + ChallengeWindowSec.
	// Batches submitted before the switch carry no submission time and keep
	// the block-count rule.
	FeatureRDKChallengeWindowTime = "rdk-challenge-window-time"
)

// V324RdkFeatures is the RDK area's share of the v3.2.4 release set.
var V324RdkFeatures = []string{FeatureRDKChallengeWindowTime}
