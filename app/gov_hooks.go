package app

import (
	"context"
	"fmt"

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

// proposalReward is what a passed proposal pays its proposer (QSR-2026-0162).
//
// It used to be a flat 1,000 QOR from the community treasury for every passed
// proposal, a hundred times the 10 QOR minimum deposit, so a trivial proposal
// that could win a vote was profitable. Bounded, the reward is the smaller of
// ProposalRewardAmount and the uqor the proposal had on deposit: it can never
// exceed the stake the proposer side put at risk of a veto burn. The reward
// stays automatic; governance raising min_deposit raises what it can reach.
func proposalReward(ctx sdk.Context, totalDeposit sdk.Coins) math.Int {
	if GovProposerRewardCapEnabled == nil || !GovProposerRewardCapEnabled(ctx) {
		return ProposalRewardAmount
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
