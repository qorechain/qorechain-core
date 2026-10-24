package types

// v3.2.5 governance and burn switches. Below the v3.2.5 plan height every one
// of them is off, so history replays byte-identically.
const (
	// FeatureGovExpeditedKeepsVotes keeps the votes of an expedited proposal
	// that fails its expedited tally (QSR-2026-0628). x/gov deletes every vote
	// as it counts it, so a failed expedited proposal converted to a regular
	// one reached its regular end with only the votes cast after the
	// conversion. With the switch the votes stay and count again at the
	// regular end (voters may still change them); they are deleted when the
	// proposal passes, at the regular end, or on cancel, as before. Full build
	// only: the switch lives in the patched SDK
	// (deps/patches/cosmos-sdk/0010).
	FeatureGovExpeditedKeepsVotes = "gov-expedited-keeps-votes"

	// FeatureBurnFailedTxConvertedFee makes the failed_tx burn channel record
	// a fee paid in a converted (non-native) denom (QSR-2026-0624 F-1,
	// QSR-2026-0633). The channel re-derived the fee from the transaction,
	// which still carries the foreign denom after conversion, so it recorded
	// nothing for it; with the switch it records the uqor the conversion paid
	// to the fee collector.
	FeatureBurnFailedTxConvertedFee = "burn-failed-tx-converted-fee"
)

// V325GovBurnFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325GovBurnFeatures = []string{
	FeatureGovExpeditedKeepsVotes,
	FeatureBurnFailedTxConvertedFee,
}
