package app

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// distributionLiability answers what the distribution module actually owes, so
// the burn module can tell that apart from what merely sits in its balance.
//
// The two are not the same, and the difference is the defect this release
// closes: a plain module-to-module transfer put coins in that balance without
// recording them anywhere the module pays from, so nobody could ever withdraw
// them. Everything the module owes IS recorded: outstanding rewards per
// validator (which delegators draw from), commission not yet withdrawn, and
// the community pool it still holds. What is left over backs no claim.
type distributionLiability struct {
	k distrkeeper.Keeper
}

// TotalOwed sums the module's recorded liabilities, truncated to whole uqor.
// Truncation is deliberate and conservative in the safe direction: the
// fractional remainders stay counted as owed rather than as releasable.
func (d distributionLiability) TotalOwed(ctx sdk.Context) math.Int {
	total := sdk.NewDecCoins()

	d.k.IterateValidatorOutstandingRewards(ctx,
		func(_ sdk.ValAddress, rewards distrtypes.ValidatorOutstandingRewards) bool {
			total = total.Add(rewards.Rewards...)
			return false
		})

	d.k.IterateValidatorAccumulatedCommissions(ctx,
		func(_ sdk.ValAddress, commission distrtypes.ValidatorAccumulatedCommission) bool {
			total = total.Add(commission.Commission...)
			return false
		})

	pool, err := d.k.FeePool.Get(ctx)
	if err == nil {
		total = total.Add(pool.CommunityPool...)
	}

	owed, _ := total.TruncateDecimal()
	return owed.AmountOf("uqor")
}
