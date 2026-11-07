package app

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// QSR-2026-0162: the proposer reward is the flat amount below the switch (the
// six historical rewards replay unchanged) and min(amount, uqor deposit) above.
func TestProposalRewardBoundedByDeposit(t *testing.T) {
	defer func(prev, own func(sdk.Context) bool) {
		GovProposerRewardCapEnabled, GovProposerRewardOwnDepositEnabled = prev, own
	}(GovProposerRewardCapEnabled, GovProposerRewardOwnDepositEnabled)
	GovProposerRewardOwnDepositEnabled = nil // the v3.2.2 bound alone
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

// From gov-proposer-reward-own-deposit the reward is bounded by the
// proposer's own uqor deposit, which the end blocker reads before it refunds
// the deposits and hands to the hook in the context (WithProposerOwnDeposit).
// Other depositors' coins no longer count, and a context without the
// annotation (the public build, whose SDK has no such step) pays nothing.
// Below the switch the bound is the total deposit, as from v3.2.2.
func TestProposalRewardBoundedByProposerOwnDeposit(t *testing.T) {
	defer func(cap, own func(sdk.Context) bool) {
		GovProposerRewardCapEnabled, GovProposerRewardOwnDepositEnabled = cap, own
	}(GovProposerRewardCapEnabled, GovProposerRewardOwnDepositEnabled)
	GovProposerRewardCapEnabled = func(sdk.Context) bool { return true }
	base := sdk.Context{}.WithContext(context.Background())
	qorInt := func(n int64) math.Int { return math.NewInt(n * 1_000_000) }
	qor := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewCoin("uqor", qorInt(n))) }

	for _, c := range []struct {
		name      string
		total     sdk.Coins
		own       *math.Int // nil: no annotation
		off, want math.Int
	}{
		{"proposer 10 QOR, another depositor 5,000 QOR", qor(5_010), ptr(qorInt(10)), ProposalRewardAmount, qorInt(10)},
		{"proposer deposited nothing", qor(5_000), ptr(math.ZeroInt()), ProposalRewardAmount, math.ZeroInt()},
		{"proposer 2,000 QOR", qor(2_000), ptr(qorInt(2_000)), ProposalRewardAmount, ProposalRewardAmount},
		{"proposer 400 QOR of 400", qor(400), ptr(qorInt(400)), qorInt(400), qorInt(400)},
		{"no annotation", qor(5_000), nil, ProposalRewardAmount, math.ZeroInt()},
	} {
		ctx := base
		if c.own != nil {
			ctx = WithProposerOwnDeposit(ctx, *c.own)
		}
		GovProposerRewardOwnDepositEnabled = func(sdk.Context) bool { return false }
		if got := proposalReward(ctx, c.total); !got.Equal(c.off) {
			t.Errorf("%s, switch off: paid %s, want %s", c.name, got, c.off)
		}
		GovProposerRewardOwnDepositEnabled = func(sdk.Context) bool { return true }
		if got := proposalReward(ctx, c.total); !got.Equal(c.want) {
			t.Errorf("%s, switch on: paid %s, want %s", c.name, got, c.want)
		}
	}
}

func ptr(v math.Int) *math.Int { return &v }
