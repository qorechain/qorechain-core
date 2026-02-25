//go:build proprietary

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// HandleMsgSetAgentMode processes a request to change the RL agent operating mode.
func (k *Keeper) HandleMsgSetAgentMode(ctx sdk.Context, msg *types.MsgSetAgentMode) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.ErrRLDisabled
	}

	// Update agent mode in params
	params.AgentMode = msg.Mode
	if err := k.SetParams(ctx, params); err != nil {
		return fmt.Errorf("failed to update agent mode: %w", err)
	}

	// Update agent status
	status := k.GetAgentStatus(ctx)
	status.Mode = msg.Mode
	if err := k.SetAgentStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	// If pausing, clear the in-memory agent
	if msg.Mode == types.AgentModePaused {
		k.agent = nil
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAgentModeChanged,
			sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute(types.AttributeKeyAgentMode, msg.Mode.String()),
		),
	)

	k.logger.Info("agent mode changed",
		"authority", msg.Authority,
		"new_mode", msg.Mode.String(),
		"height", ctx.BlockHeight(),
	)

	return nil
}

// HandleMsgResumeAgent processes a request to resume the RL agent from paused state.
func (k *Keeper) HandleMsgResumeAgent(ctx sdk.Context, msg *types.MsgResumeAgent) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.ErrRLDisabled
	}

	if params.AgentMode != types.AgentModePaused {
		return fmt.Errorf("agent is not paused (current mode: %s)", params.AgentMode.String())
	}

	// Resume to shadow mode
	params.AgentMode = types.AgentModeShadow
	if err := k.SetParams(ctx, params); err != nil {
		return fmt.Errorf("failed to resume agent: %w", err)
	}

	status := k.GetAgentStatus(ctx)
	status.Mode = types.AgentModeShadow
	status.CircuitBreakerActive = false
	status.BlocksSinceRevert = 0
	if err := k.SetAgentStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAgentModeChanged,
			sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute(types.AttributeKeyAgentMode, types.AgentModeShadow.String()),
		),
	)

	k.logger.Info("agent resumed to shadow mode",
		"authority", msg.Authority,
		"height", ctx.BlockHeight(),
	)

	return nil
}

// HandleMsgUpdatePolicy processes a request to replace the current policy weights.
func (k *Keeper) HandleMsgUpdatePolicy(ctx sdk.Context, msg *types.MsgUpdatePolicy) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.ErrRLDisabled
	}

	pw := &msg.Weights
	pw.UpdatedAt = ctx.BlockHeight()

	if err := k.SetPolicyWeights(ctx, pw); err != nil {
		return fmt.Errorf("failed to store policy weights: %w", err)
	}

	// Recreate agent with new weights
	k.agent = NewPPOAgent(pw)

	// Update agent status epoch
	status := k.GetAgentStatus(ctx)
	status.CurrentEpoch = pw.Epoch
	if err := k.SetAgentStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePolicyUpdated,
			sdk.NewAttribute(types.AttributeKeyHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute(types.AttributeKeyEpoch, fmt.Sprintf("%d", pw.Epoch)),
		),
	)

	k.logger.Info("policy weights updated",
		"authority", msg.Authority,
		"epoch", pw.Epoch,
		"total_params", pw.Config.TotalParams(),
		"height", ctx.BlockHeight(),
	)

	return nil
}

// HandleMsgUpdateRewardWeights processes a request to change the reward function weighting.
func (k *Keeper) HandleMsgUpdateRewardWeights(ctx sdk.Context, msg *types.MsgUpdateRewardWeights) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.ErrRLDisabled
	}

	params.RewardWeights = msg.Weights
	if err := k.SetParams(ctx, params); err != nil {
		return fmt.Errorf("failed to update reward weights: %w", err)
	}

	k.logger.Info("reward weights updated",
		"authority", msg.Authority,
		"throughput", msg.Weights.Throughput,
		"finality", msg.Weights.Finality,
		"decentralization", msg.Weights.Decentralization,
		"mev", msg.Weights.MEV,
		"failed_txs", msg.Weights.FailedTxs,
		"height", ctx.BlockHeight(),
	)

	return nil
}
