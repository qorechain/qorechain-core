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
// governance proposal passes.
var ProposalRewardAmount = math.NewInt(1_000_000_000) // 1,000 QOR

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

	reward := sdk.NewCoins(sdk.NewCoin("uqor", ProposalRewardAmount))
	if err := h.bankKeeper.SendCoinsFromModuleToAccount(ctx, "protocolpool", proposerAddr, reward); err != nil {
		// Log but don't fail the EndBlocker — treasury may be empty
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Warn("governance proposal reward: failed to send reward",
			"proposal_id", proposalID,
			"proposer", proposal.Proposer,
			"error", err,
		)
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"proposal_reward",
			sdk.NewAttribute("proposal_id", fmt.Sprintf("%d", proposalID)),
			sdk.NewAttribute("proposer", proposal.Proposer),
			sdk.NewAttribute("reward_amount", ProposalRewardAmount.String()),
		),
	)

	return nil
}
