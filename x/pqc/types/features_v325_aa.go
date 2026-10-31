package types

// v3.2.5 abstract-account area. Below the v3.2.5 plan height both are off, so
// history replays byte-identically: rules are saved and enforced exactly as
// before and the module configuration has no message that changes it.
const (
	// FeatureAASpendingRuleBounds closes QSR-2026-0611. When a rule is saved
	// (MsgUpdateSpendingRules) the message server refuses an enabled rule that
	// bounds nothing, a denom_limits entry whose per_tx and daily amounts are
	// both empty or zero, and an erc20/, erc721/ or erc1155/ denom whose
	// contract address does not parse. The spending check treats an all-zero
	// denom_limits entry as absent, so the rule's per_tx_limit and daily_limit
	// apply to that denom; this covers rules stored before the switch.
	FeatureAASpendingRuleBounds = "abstractaccount-spending-rule-bounds"

	// FeatureAAConfigGovernance routes MsgUpdateConfig: governance replaces the
	// module configuration (including the enabled flag that gates account
	// creation, authenticator registration and both execute lanes). Before it
	// no message could change the stored configuration.
	FeatureAAConfigGovernance = "abstractaccount-config-governance"
)

// V325AAFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325AAFeatures = []string{FeatureAASpendingRuleBounds, FeatureAAConfigGovernance}
