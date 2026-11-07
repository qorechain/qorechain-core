package types

// v3.2.8 reward accounting fixes. Below the v3.2.8 plan height all four are
// off, so history replays byte-identically: the same bonus, reward and
// governance amounts, the same state writes and the same events.
const (
	// FeatureStakingTierExpiredLockOwnShares changes how a settlement shares a
	// delegation among its staking tier locks. Only the locks still open at
	// the settlement time share the delegation's current shares, oldest first;
	// a lock that has expired (released by the end blocker, or past its end
	// and not yet released) accrues its last part, up to its end time, on its
	// own shares, which were locked and so fully backed for that whole time,
	// and it is paid even when the holder has since unbonded the rest of the
	// delegation. Before it every lock of the delegation, expired ones
	// included, was served oldest first against the current shares, so a
	// holder who unbonded the stake an expired lock had freed lost the bonus
	// of the lock still open (50M QOR locked, 70M delegated after unbonding
	// 50M of 120M: the open lock accrued on 20M), and an expired lock whose
	// delegation was fully unbonded lost its last part.
	FeatureStakingTierExpiredLockOwnShares = "stakingtier-expired-lock-own-shares"

	// FeatureStakingTierBaseNetOfCommunityTax makes the base yield a
	// settlement computes (and records as last_base_apy_bps, which the APR
	// queries read) net of the distribution community tax: the 47% of
	// emission that reaches stakers through fee_collector is split by
	// x/distribution only after it takes community_tax (2%). Before it the
	// base was computed on the whole 47%, so it read about 2% (relative)
	// higher than stakers receive, and the bonus room under
	// max_total_apy_bps was cut by the same amount.
	FeatureStakingTierBaseNetOfCommunityTax = "stakingtier-base-net-of-community-tax"

	// FeatureLightnodeMonthlyElapsedClamp clamps the time a light node's
	// monthly reward bound is prorated over to one reward month (30 days).
	// Before it the bound was max_reward_per_month_per_node times the seconds
	// since the previous settlement over 30 days with no ceiling, so a
	// settlement after a gap longer than a month (a halt, or a long dry
	// spell) could credit one node more than a month's maximum at once.
	FeatureLightnodeMonthlyElapsedClamp = "lightnode-monthly-elapsed-clamp"

	// FeatureGovProposerRewardOwnDeposit bounds the reward a passed
	// governance proposal pays its proposer by the proposer's own uqor
	// deposit on that proposal, read just before the deposits are refunded.
	// Before it the bound was the proposal's total deposit, which counts
	// other depositors' coins and is refunded on pass anyway, so a proposer
	// who put in nothing could still be paid up to 1,000 QOR.
	FeatureGovProposerRewardOwnDeposit = "gov-proposer-reward-own-deposit"
)

// V328EconFeatures are the v3.2.8 switches for staking tier, light node reward and governance reward accounting.
var V328EconFeatures = []string{
	FeatureStakingTierExpiredLockOwnShares,
	FeatureStakingTierBaseNetOfCommunityTax,
	FeatureLightnodeMonthlyElapsedClamp,
	FeatureGovProposerRewardOwnDeposit,
}
