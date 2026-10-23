package types

// v3.2.4 staking economics. Below the v3.2.4 plan height both are off, so
// history replays byte-identically: no new state is written, no gas, event or
// emission changes.
const (
	// x/stakingtier: delegation locks with a bonus yield paid per inflation
	// epoch from the stakingtier module account, minted by x/inflation under
	// the same cumulative emission cap, and the lock enforced on every
	// undelegation and redelegation in the staking message server.
	FeatureStakingTiers = "staking-tiers"

	// x/inflation: the "target_apy" emission mode, which mints per epoch what
	// the bonded stake needs to earn target_apy_bps through the fee split.
	FeatureInflationTargetAPY = "inflation-target-apy"
)

// V324StakingTierFeatures is this area's share of the v3.2.4 release set.
var V324StakingTierFeatures = []string{FeatureStakingTiers, FeatureInflationTargetAPY}
