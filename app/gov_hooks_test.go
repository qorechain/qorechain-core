package app

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QSR-2026-0162: the proposer reward is the flat amount below the switch (the
// six historical rewards replay unchanged) and min(amount, uqor deposit) above.
func TestProposalRewardBoundedByDeposit(t *testing.T) {
	defer func(prev func(sdk.Context) bool) { GovProposerRewardCapEnabled = prev }(GovProposerRewardCapEnabled)
	ctx := sdk.Context{}
	qor := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewCoin("uqor", math.NewInt(n*1_000_000))) }

	GovProposerRewardCapEnabled = nil
	if got := proposalReward(ctx, qor(10)); !got.Equal(ProposalRewardAmount) {
		t.Fatalf("unwired: paid %s, want the flat %s", got, ProposalRewardAmount)
	}
	GovProposerRewardCapEnabled = func(sdk.Context) bool { return false }
	if got := proposalReward(ctx, qor(10)); !got.Equal(ProposalRewardAmount) {
		t.Fatalf("switch off: paid %s, want the flat %s", got, ProposalRewardAmount)
	}

	GovProposerRewardCapEnabled = func(sdk.Context) bool { return true }
	for _, c := range []struct {
		deposit sdk.Coins
		want    math.Int
	}{
		{qor(10), math.NewInt(10_000_000)},                                       // the 10 QOR minimum deposit
		{qor(1_000), ProposalRewardAmount},                                       // exactly the amount
		{qor(5_000), ProposalRewardAmount},                                       // more than the amount: capped
		{sdk.NewCoins(), math.ZeroInt()},                                         // nothing on deposit
		{sdk.NewCoins(sdk.NewInt64Coin("uatom", 5_000_000_000)), math.ZeroInt()}, // not uqor
	} {
		if got := proposalReward(ctx, c.deposit); !got.Equal(c.want) {
			t.Errorf("deposit %s: paid %s, want %s", c.deposit, got, c.want)
		}
	}
}
