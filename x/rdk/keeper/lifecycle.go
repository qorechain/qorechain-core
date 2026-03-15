//go:build proprietary

package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	burntypes "github.com/qorechain/qorechain-core/x/burn/types"
	multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"
	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// CreateRollup validates, escrows stake, burns creation fee, registers layer, and activates.
func (k Keeper) CreateRollup(ctx sdk.Context, config types.RollupConfig) (*types.RollupConfig, error) {
	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Check max rollups
	params := k.GetParams(ctx)
	if k.rollupCount(ctx) >= params.MaxRollups {
		return nil, types.ErrMaxRollupsReached
	}

	// Check duplicate
	if _, err := k.GetRollup(ctx, config.RollupID); err == nil {
		return nil, types.ErrRollupAlreadyExists
	}

	// Check min stake
	if config.StakeAmount < params.MinStakeForRollup {
		return nil, types.ErrInsufficientStake
	}

	// Escrow QOR via bank: send from creator to module
	creatorAddr, err := sdk.AccAddressFromBech32(config.Creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address: %w", err)
	}
	stakeCoins := sdk.NewCoins(sdk.NewCoin("uqor", math.NewInt(config.StakeAmount)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, stakeCoins); err != nil {
		return nil, fmt.Errorf("failed to escrow stake: %w", err)
	}

	// Burn creation fee (non-fatal)
	burnRate := parseDec(params.RollupCreationBurnRate)
	if burnRate.GT(math.LegacyZeroDec()) {
		burnAmount := math.LegacyNewDecFromInt(math.NewInt(config.StakeAmount)).Mul(burnRate).TruncateInt()
		if burnAmount.IsPositive() {
			if burnErr := k.burnKeeper.BurnFromSource(ctx, burntypes.BurnSourceRollupCreate, burnAmount, config.RollupID); burnErr != nil {
				k.logger.Warn("rollup creation burn failed (non-fatal)", "error", burnErr, "rollup_id", config.RollupID)
			}
		}
	}

	// Register as a layer in x/multilayer
	layerMsg := &multilayertypes.MsgRegisterSidechain{
		Creator:                  config.Creator,
		LayerID:                  config.RollupID,
		Description:              fmt.Sprintf("RDK rollup: %s (%s)", config.RollupID, config.Profile),
		TargetBlockTimeMs:        config.BlockTimeMs,
		MaxTransactionsPerBlock:  config.MaxTxPerBlock,
		MinValidators:            1,
		SettlementIntervalBlocks: 100,
		SupportedVMTypes:         []string{config.VMType},
		SupportedDomains:         []string{string(config.Profile)},
	}
	resp, err := k.multilayerKeeper.RegisterSidechain(ctx, layerMsg)
	if err != nil {
		k.logger.Warn("layer registration failed (non-fatal)", "error", err, "rollup_id", config.RollupID)
	}

	// Set status and metadata
	config.Status = types.RollupStatusActive
	if resp != nil {
		config.LayerID = resp.LayerID
	} else {
		config.LayerID = config.RollupID
	}
	config.CreatedHeight = ctx.BlockHeight()
	config.CreatedAt = ctx.BlockTime().UTC()

	if err := k.setRollup(ctx, config); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventRollupCreated,
		sdk.NewAttribute("rollup_id", config.RollupID),
		sdk.NewAttribute("creator", config.Creator),
		sdk.NewAttribute("profile", string(config.Profile)),
		sdk.NewAttribute("settlement_mode", string(config.SettlementMode)),
	))

	k.logger.Info("rollup created", "rollup_id", config.RollupID, "creator", config.Creator, "profile", config.Profile)
	return &config, nil
}

// PauseRollup pauses a rollup. Only the creator can pause.
func (k Keeper) PauseRollup(ctx sdk.Context, rollupID string, reason string) error {
	config, err := k.GetRollup(ctx, rollupID)
	if err != nil {
		return err
	}
	if config.Status != types.RollupStatusActive {
		return types.ErrRollupNotActive
	}

	config.Status = types.RollupStatusPaused
	if err := k.setRollup(ctx, *config); err != nil {
		return err
	}

	// Update layer status in multilayer (non-fatal)
	if updateErr := k.multilayerKeeper.UpdateLayerStatus(ctx, config.LayerID, multilayertypes.LayerStatusSuspended, reason); updateErr != nil {
		k.logger.Warn("failed to update layer status (non-fatal)", "error", updateErr)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventRollupPaused,
		sdk.NewAttribute("rollup_id", rollupID),
		sdk.NewAttribute("reason", reason),
	))

	k.logger.Info("rollup paused", "rollup_id", rollupID, "reason", reason)
	return nil
}

// ResumeRollup resumes a paused rollup. Only the creator can resume.
func (k Keeper) ResumeRollup(ctx sdk.Context, rollupID string) error {
	config, err := k.GetRollup(ctx, rollupID)
	if err != nil {
		return err
	}
	if config.Status != types.RollupStatusPaused {
		return fmt.Errorf("rollup must be paused to resume, current status: %s", config.Status)
	}

	config.Status = types.RollupStatusActive
	if err := k.setRollup(ctx, *config); err != nil {
		return err
	}

	// Update layer status in multilayer (non-fatal)
	if updateErr := k.multilayerKeeper.UpdateLayerStatus(ctx, config.LayerID, multilayertypes.LayerStatusActive, "resumed"); updateErr != nil {
		k.logger.Warn("failed to update layer status (non-fatal)", "error", updateErr)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventRollupResumed,
		sdk.NewAttribute("rollup_id", rollupID),
	))

	k.logger.Info("rollup resumed", "rollup_id", rollupID)
	return nil
}

// StopRollup permanently stops a rollup and returns the bond.
func (k Keeper) StopRollup(ctx sdk.Context, rollupID string) error {
	config, err := k.GetRollup(ctx, rollupID)
	if err != nil {
		return err
	}
	if config.Status == types.RollupStatusStopped {
		return fmt.Errorf("rollup already stopped")
	}

	// Return bond to creator
	creatorAddr, addrErr := sdk.AccAddressFromBech32(config.Creator)
	if addrErr == nil && config.StakeAmount > 0 {
		returnCoins := sdk.NewCoins(sdk.NewCoin("uqor", math.NewInt(config.StakeAmount)))
		if sendErr := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAddr, returnCoins); sendErr != nil {
			k.logger.Warn("failed to return bond (non-fatal)", "error", sendErr, "rollup_id", rollupID)
		}
	}

	config.Status = types.RollupStatusStopped
	if err := k.setRollup(ctx, *config); err != nil {
		return err
	}

	// Update layer status in multilayer (non-fatal)
	if updateErr := k.multilayerKeeper.UpdateLayerStatus(ctx, config.LayerID, multilayertypes.LayerStatusDecommissioned, "stopped"); updateErr != nil {
		k.logger.Warn("failed to update layer status (non-fatal)", "error", updateErr)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventRollupStopped,
		sdk.NewAttribute("rollup_id", rollupID),
	))

	k.logger.Info("rollup stopped", "rollup_id", rollupID)
	return nil
}

// parseDec parses a decimal string, returning zero on error.
func parseDec(s string) math.LegacyDec {
	d, err := math.LegacyNewDecFromStr(s)
	if err != nil {
		return math.LegacyZeroDec()
	}
	return d
}
