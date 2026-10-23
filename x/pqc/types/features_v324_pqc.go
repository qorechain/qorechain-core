package types

// v3.2.4 post-quantum key switches. Below the v3.2.4 plan height every one of
// them is off, so history replays byte-identically.
const (
	// FeaturePQCPossessionFlag records on the account whether its current key
	// was proven by a proof of possession: a MsgRegisterPQCKeyV2 carrying a
	// valid proof_of_possession, or a rotation or migration (both verify a
	// signature by the new key over bytes naming the chain, the account and
	// the key). A registration without a proof records false, so a key that
	// changes without one clears the flag. Proof of possession stays optional
	// for registration. The account query and the pqcKeyStatus precompile
	// expose the flag.
	FeaturePQCPossessionFlag = "pqc-possession-flag"
)

// V324PqcFeatures is the post-quantum area's share of the v3.2.4 release set.
var V324PqcFeatures = []string{FeaturePQCPossessionFlag}
