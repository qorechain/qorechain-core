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

// TotalOwed is the liability as the v3.2.0 release handler computed it:
// outstanding rewards, plus unwithdrawn commission, plus the community pool.
//
// It overstates the liability (QSR-2026-0341): the SDK credits a validator's
// full allocation, commission included, to its outstanding rewards and also
// records the commission slice in the accumulated commission, so adding both
// counts the commission twice. It is kept byte-for-byte because the v3.2.0
// upgrade handler calls it, and a node replaying that height with this binary
// must record the same snapshot. New code uses ClaimsOwed.
func (d distributionLiability) TotalOwed(ctx sdk.Context) math.Int {
	total := d.outstandingAndPool(ctx)

	d.k.IterateValidatorAccumulatedCommissions(ctx,
		func(_ sdk.ValAddress, commission distrtypes.ValidatorAccumulatedCommission) bool {
			total = total.Add(commission.Commission...)
			return false
		})

	owed, _ := total.TruncateDecimal()
	return owed.AmountOf("uqor")
}

// ClaimsOwed is what the module really owes, truncated to whole uqor: the
// outstanding rewards (which already contain the unwithdrawn commission) and
// the community pool it still holds. Truncation is deliberate and conservative
// in the safe direction: the fractional remainders stay counted as owed rather
// than as releasable.
func (d distributionLiability) ClaimsOwed(ctx sdk.Context) math.Int {
	owed, _ := d.outstandingAndPool(ctx).TruncateDecimal()
	return owed.AmountOf("uqor")
}

func (d distributionLiability) outstandingAndPool(ctx sdk.Context) sdk.DecCoins {
	total := sdk.NewDecCoins()

	d.k.IterateValidatorOutstandingRewards(ctx,
		func(_ sdk.ValAddress, rewards distrtypes.ValidatorOutstandingRewards) bool {
			total = total.Add(rewards.Rewards...)
			return false
		})

	pool, err := d.k.FeePool.Get(ctx)
	if err == nil {
		total = total.Add(pool.CommunityPool...)
	}
	return total
}
