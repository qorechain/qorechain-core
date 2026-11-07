package app

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

var _ govtypes.GovHooks = ProposalRewardHook{}

// ProposalRewardAmount is the reward (in uqor) sent to the proposer when a
// governance proposal passes. From the v3.2.2 plan height it is a ceiling: see
// proposalReward.
var ProposalRewardAmount = math.NewInt(1_000_000_000) // 1,000 QOR

// GovProposerRewardCapEnabled answers, per block and from chain state, whether
// the proposer reward is bounded by the proposal's deposit
// (FeatureGovProposerRewardCap, v3.2.2). Unset or false pays the flat
// historical amount, so the rewards already paid replay unchanged. Wired by the
// application to the protocol-feature record.
var GovProposerRewardCapEnabled func(ctx sdk.Context) bool

// GovProposerRewardOwnDepositEnabled answers, per block and from chain state,
// whether the proposer reward is bounded by the proposer's own deposit
// (FeatureGovProposerRewardOwnDeposit, v3.2.8) instead of the proposal's total
// deposit. Unset or false keeps the v3.2.2 bound. Wired by the application to
// the protocol-feature record.
var GovProposerRewardOwnDepositEnabled func(ctx sdk.Context) bool

// proposerOwnDepositKey carries the proposer's own uqor deposit from the gov
// end blocker, which reads it before the deposits are refunded, to the
// AfterProposalVotingPeriodEnded hook, which runs after they are deleted.
type proposerOwnDepositKey struct{}

// WithProposerOwnDeposit annotates the context the reward hook of one
// proposal runs in with the proposer's own uqor deposit on it.
func WithProposerOwnDeposit(ctx sdk.Context, amount math.Int) sdk.Context {
	return ctx.WithValue(proposerOwnDepositKey{}, amount)
}

// proposerOwnDepositFrom reads what WithProposerOwnDeposit stored.
func proposerOwnDepositFrom(ctx sdk.Context) (math.Int, bool) {
	v, ok := ctx.Value(proposerOwnDepositKey{}).(math.Int)
	if !ok || v.IsNil() {
		return math.ZeroInt(), false
	}
	return v, true
}

// ProposerOwnDeposit is the uqor the proposer has on deposit on proposal,
// zero when they deposited nothing. Only meaningful before the end blocker
// refunds or burns the deposits.
func ProposerOwnDeposit(ctx sdk.Context, gk *govkeeper.Keeper, proposal govv1.Proposal) math.Int {
	proposer, err := sdk.AccAddressFromBech32(proposal.Proposer)
	if err != nil || gk == nil {
		return math.ZeroInt()
	}
	dep, err := gk.Deposits.Get(ctx, collections.Join(proposal.Id, proposer))
	if err != nil {
		return math.ZeroInt()
	}
	return sdk.Coins(dep.Amount).AmountOf("uqor")
}

// proposalReward is what a passed proposal pays its proposer.
//
// It used to be a flat 1,000 QOR from the community treasury for every passed
// proposal, a hundred times the 10 QOR minimum deposit, so a trivial proposal
// that could win a vote was profitable. From the v3.2.2 cap the reward is the
// smaller of ProposalRewardAmount and the uqor the proposal had on deposit.
// That total counts other depositors' coins, and every deposit is refunded
// when a proposal passes, so it did not bound the reward by anything the
// proposer risked: a proposer who put in nothing was still paid up to the
// full amount. From the v3.2.8 switch the bound is the proposer's own uqor
// deposit, read by the end blocker before the refund (deps/patches/cosmos-sdk
// 0011 in the full build). A context without that reading pays nothing. The
// reward stays automatic; governance raising min_deposit raises what it can
// reach.
func proposalReward(ctx sdk.Context, totalDeposit sdk.Coins) math.Int {
	if GovProposerRewardCapEnabled == nil || !GovProposerRewardCapEnabled(ctx) {
		return ProposalRewardAmount
	}
	if GovProposerRewardOwnDepositEnabled != nil && GovProposerRewardOwnDepositEnabled(ctx) {
		own, _ := proposerOwnDepositFrom(ctx)
		return math.MinInt(ProposalRewardAmount, own)
	}
	return math.MinInt(ProposalRewardAmount, totalDeposit.AmountOf("uqor"))
}

// ProposalRewardHook rewards proposers whose governance proposals pass.
type ProposalRewardHook struct {
	govKeeper  *govkeeper.Keeper
	bankKeeper bankkeeper.BaseKeeper
}

func NewProposalRewardHook(govKeeper *govkeeper.Keeper, bankKeeper bankkeeper.BaseKeeper) ProposalRewardHook {
	return ProposalRewardHook{
		govKeeper:  govKeeper,
		bankKeeper: bankKeeper,
	}
}

// ProvideGovHooks wires ProposalRewardHook into the gov keeper through depinject.
//
// Under the depinject (v0.53) flow the gov keeper's hooks are installed during
// appBuilder.Build() by the gov module's InvokeSetHooks, which collects every
// GovHooksWrapper provided one-per-module. Because that runs unconditionally
// (setting an empty MultiGovHooks when no wrappers exist), calling
// GovKeeper.SetHooks() manually afterwards panics with "cannot set governance
// hooks twice". Providing the wrapper here instead lets depinject install our
// hook as part of that same pass.
func ProvideGovHooks(govKeeper *govkeeper.Keeper, bankKeeper bankkeeper.BaseKeeper) govtypes.GovHooksWrapper {
	return govtypes.GovHooksWrapper{GovHooks: NewProposalRewardHook(govKeeper, bankKeeper)}
}

func (h ProposalRewardHook) AfterProposalSubmission(_ context.Context, _ uint64) error { return nil }
func (h ProposalRewardHook) AfterProposalDeposit(_ context.Context, _ uint64, _ sdk.AccAddress) error {
	return nil
}
func (h ProposalRewardHook) AfterProposalVote(_ context.Context, _ uint64, _ sdk.AccAddress) error {
	return nil
}
func (h ProposalRewardHook) AfterProposalFailedMinDeposit(_ context.Context, _ uint64) error {
	return nil
}

// AfterProposalVotingPeriodEnded is called after the voting period ends and
// the proposal status has been determined. If the proposal passed, it sends
// ProposalRewardAmount from the protocol pool to the proposer.
func (h ProposalRewardHook) AfterProposalVotingPeriodEnded(ctx context.Context, proposalID uint64) error {
	proposal, err := h.govKeeper.Proposals.Get(ctx, proposalID)
	if err != nil {
		return nil // proposal not found — nothing to do
	}

	if proposal.Status != govv1.StatusPassed {
		return nil
	}

	proposerAddr, err := sdk.AccAddressFromBech32(proposal.Proposer)
	if err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	amount := proposalReward(sdkCtx, proposal.TotalDeposit)
	if !amount.IsPositive() {
		sdkCtx.Logger().Info("governance proposal reward: nothing to pay, the proposal had no uqor on deposit",
			"proposal_id", proposalID, "proposer", proposal.Proposer)
		return nil
	}
	reward := sdk.NewCoins(sdk.NewCoin("uqor", amount))
	if err := h.bankKeeper.SendCoinsFromModuleToAccount(ctx, "protocolpool", proposerAddr, reward); err != nil {
		// Log but don't fail the EndBlocker — treasury may be empty
		sdkCtx.Logger().Warn("governance proposal reward: failed to send reward",
			"proposal_id", proposalID,
			"proposer", proposal.Proposer,
			"error", err,
		)
		return nil
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"proposal_reward",
			sdk.NewAttribute("proposal_id", fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute("proposer", proposal.Proposer),
			sdk.NewAttribute("reward_amount", amount.String()),
		),
	)

	return nil
}
