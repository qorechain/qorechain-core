package types

// v3.2.4 bridge switch. Below the v3.2.4 plan height it is off, so history
// replays byte-identically.
const (
	// FeatureBridgeValidatorKeyPoP makes MsgRegisterBridgeValidator prove
	// possession of its PQC public key (proof_of_possession, the x/pqc proof
	// format) and refuses a key another bridge validator already uses.
	FeatureBridgeValidatorKeyPoP = "bridge-validator-key-pop"
)

// V324BridgeFeatures is the bridge area's share of the v3.2.4 release set.
var V324BridgeFeatures = []string{FeatureBridgeValidatorKeyPoP}
