package types

// v3.2.4 light node reward window. Below the v3.2.4 plan height it is off, so
// history replays byte-identically.
const (
	// A node's reward bound counts from the later of the pool's previous
	// settlement and the node's own registration, so a node is never paid for
	// time before it existed.
	FeatureLightnodeRegistrationWindow = "lightnode-registration-window"
)

// V324LightnodeFeatures is this area's share of the v3.2.4 release set.
var V324LightnodeFeatures = []string{FeatureLightnodeRegistrationWindow}
