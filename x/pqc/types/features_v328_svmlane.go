package types

// v3.2.8: the SVM lane becomes a governance decision on the public testnet
// only.
//
// Until this switch the lane is closed by a compile-time constant in the SVM
// keeper (SVMLaneHardDisabled): GetParams reports Enabled=false whatever the
// stored params say, so no transaction, governance included, can open it, and
// every change of mind costs a binary release. With the switch on, the stored
// Enabled param decides, and x/svm MsgUpdateParams (governance only) opens and
// closes the lane.
//
// CHAIN-SCOPED, NOT RELEASED. The feature is deliberately absent from
// ReleaseV3_2_8ProtocolFeatures and AllProtocolFeatures: a chain started on
// this binary does not get it at genesis, and the v3.2.8 release handler
// enables it only on SVMLaneGovernedChainID. On every other chain, mainnet
// included, the compile-time switch keeps the lane closed exactly as before,
// even if governance stores enabled=true.
//
// CLOSED ON ARRIVAL. In the same step that enables the feature, the handler
// writes the stored SVM params with Enabled=false (the live networks store
// enabled=true, a leftover of the default), so the lane is still closed in the
// block after the upgrade and opens only when a later governance proposal sets
// enabled=true. The handler also sets a finite custody cap (see
// SVMLaneGovernedInitialCustodyCapUqor).
//
// While the switch is on, the SVM custody cap param (max_custody_uqor) is
// enforced on every transfer into the svm module account.
const FeatureSVMLaneGoverned = "svm-lane-governed"

// SVMLaneGovernedChainID is the only chain on which the v3.2.8 handler enables
// FeatureSVMLaneGoverned: the public testnet.
const SVMLaneGovernedChainID = "qorechain-diana"

// SVMLaneGovernedInitialCustodyCapUqor is the custody cap the v3.2.8 handler
// writes on SVMLaneGovernedChainID: 10,000 QOR. It bounds the value an SVM
// runtime fault could reach during the soak; governance raises it with the
// same MsgUpdateParams that opens the lane.
const SVMLaneGovernedInitialCustodyCapUqor uint64 = 10_000 * 1_000_000

// ChainScopedProtocolFeatures are known to this binary and enabled only by a
// release handler on a named chain, never at genesis and never on every chain.
var ChainScopedProtocolFeatures = []string{
	FeatureSVMLaneGoverned,
}
