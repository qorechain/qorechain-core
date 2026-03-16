//go:build proprietary

package keeper

import (
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// HandleBridgeDeposit processes a bridge deposit (mint bridged assets on QoreChain).
func (k Keeper) HandleBridgeDeposit(ctx sdk.Context, msg types.MsgBridgeDeposit) (*types.BridgeOperation, error) {
	// 1. Validate chain is supported and active
	chainConfig, found := k.GetChainConfig(ctx, msg.SourceChain)
	if !found {
		return nil, types.ErrChainNotSupported.Wrapf("chain %s not configured", msg.SourceChain)
	}
	if chainConfig.Status == types.BridgeStatusPaused {
		return nil, types.ErrChainPaused.Wrapf("bridge for %s is paused", msg.SourceChain)
	}

	// 2. Validate asset is supported
	assetSupported := false
	for _, a := range chainConfig.SupportedAssets {
		if a == msg.Asset {
			assetSupported = true
			break
		}
	}
	if !assetSupported {
		return nil, types.ErrAssetNotSupported.Wrapf("asset %s not supported on %s", msg.Asset, msg.SourceChain)
	}

	// 3. Check circuit breaker
	amount, err := types.ParseAmount(msg.Amount)
	if err != nil {
		return nil, err
	}
	if err := k.CheckCircuitBreakerLimits(ctx, msg.SourceChain, amount); err != nil {
		return nil, err
	}

	// 4. Create bridge operation
	opID := k.NextOperationID(ctx)
	now := ctx.BlockTime()
	op := types.BridgeOperation{
		ID:            opID,
		Type:          types.OpTypeDeposit,
		SourceChain:   msg.SourceChain,
		DestChain:     "qorechain",
		Sender:        msg.Sender,
		Receiver:      msg.Sender, // For deposits, receiver is the QoreChain sender
		Asset:         msg.Asset,
		Amount:        msg.Amount,
		SourceTxHash:  msg.SourceTxHash,
		Status:        types.OpStatusPending,
		Attestations:  []types.Attestation{},
		PQCCommitment: msg.PQCCommitment,
		CreatedAt:     now,
	}

	// 5. Check if large transfer requires challenge period
	config := k.GetConfig(ctx)
	threshold, _ := types.ParseAmount(config.LargeTransferThreshold)
	if amount.GT(threshold) {
		challengeEnd := now.Add(time.Duration(config.ChallengePeriodSecs) * time.Second)
		op.ChallengeEndTime = &challengeEnd
	}

	if err := k.SetOperation(ctx, op); err != nil {
		return nil, err
	}

	// 6. Update locked amounts
	locked := k.GetLockedAmount(ctx, msg.SourceChain, msg.Asset)
	currentLocked, _ := types.ParseAmount(locked.TotalLocked)
	locked.TotalLocked = currentLocked.Add(amount).String()
	if err := k.SetLockedAmount(ctx, locked); err != nil {
		return nil, err
	}

	// 7. Update circuit breaker daily counter
	k.IncrementDailyUsage(ctx, msg.SourceChain, amount)

	// 8. Emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeBridgeDeposit,
			sdk.NewAttribute(types.AttributeKeyOperationID, opID),
			sdk.NewAttribute(types.AttributeKeyChain, msg.SourceChain),
			sdk.NewAttribute(types.AttributeKeyAsset, msg.Asset),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount),
			sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
		),
	})

	k.Logger().Info("bridge deposit created",
		"operation_id", opID,
		"chain", msg.SourceChain,
		"asset", msg.Asset,
		"amount", msg.Amount,
	)

	return &op, nil
}

// HandleBridgeWithdraw processes a bridge withdrawal (burn on QoreChain, trigger unlock on external chain).
func (k Keeper) HandleBridgeWithdraw(ctx sdk.Context, msg types.MsgBridgeWithdraw) (*types.BridgeOperation, error) {
	// 1. Validate chain is supported and active
	chainConfig, found := k.GetChainConfig(ctx, msg.DestinationChain)
	if !found {
		return nil, types.ErrChainNotSupported.Wrapf("chain %s not configured", msg.DestinationChain)
	}
	if chainConfig.Status == types.BridgeStatusPaused {
		return nil, types.ErrChainPaused.Wrapf("bridge for %s is paused", msg.DestinationChain)
	}

	// 2. Validate asset
	assetSupported := false
	for _, a := range chainConfig.SupportedAssets {
		if a == msg.Asset {
			assetSupported = true
			break
		}
	}
	if !assetSupported {
		return nil, types.ErrAssetNotSupported.Wrapf("asset %s not supported on %s", msg.Asset, msg.DestinationChain)
	}

	// 3. Check circuit breaker
	amount, err := types.ParseAmount(msg.Amount)
	if err != nil {
		return nil, err
	}
	if err := k.CheckCircuitBreakerLimits(ctx, msg.DestinationChain, amount); err != nil {
		return nil, err
	}

	// 4. Verify sufficient minted balance exists
	locked := k.GetLockedAmount(ctx, msg.DestinationChain, msg.Asset)
	currentMinted, _ := types.ParseAmount(locked.TotalMinted)
	if amount.GT(currentMinted) {
		return nil, types.ErrInvalidAmount.Wrapf("insufficient minted balance: have %s, need %s", currentMinted.String(), amount.String())
	}

	// 5. Create withdrawal operation
	opID := k.NextOperationID(ctx)
	now := ctx.BlockTime()
	op := types.BridgeOperation{
		ID:          opID,
		Type:        types.OpTypeWithdrawal,
		SourceChain: "qorechain",
		DestChain:   msg.DestinationChain,
		Sender:      msg.Sender,
		Receiver:    msg.DestinationAddress,
		Asset:       msg.Asset,
		Amount:      msg.Amount,
		Status:      types.OpStatusPending,
		Attestations: []types.Attestation{},
		CreatedAt:   now,
	}

	// 6. Challenge period for large withdrawals
	config := k.GetConfig(ctx)
	threshold, _ := types.ParseAmount(config.LargeTransferThreshold)
	if amount.GT(threshold) {
		challengeEnd := now.Add(time.Duration(config.ChallengePeriodSecs) * time.Second)
		op.ChallengeEndTime = &challengeEnd
	}

	if err := k.SetOperation(ctx, op); err != nil {
		return nil, err
	}

	// 7. Update circuit breaker daily counter
	k.IncrementDailyUsage(ctx, msg.DestinationChain, amount)

	// 8. Emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeBridgeWithdraw,
			sdk.NewAttribute(types.AttributeKeyOperationID, opID),
			sdk.NewAttribute(types.AttributeKeyChain, msg.DestinationChain),
			sdk.NewAttribute(types.AttributeKeyAsset, msg.Asset),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount),
			sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, msg.DestinationAddress),
		),
	})

	k.Logger().Info("bridge withdrawal created",
		"operation_id", opID,
		"chain", msg.DestinationChain,
		"asset", msg.Asset,
		"amount", msg.Amount,
	)

	return &op, nil
}

// HandleRegisterBridgeValidator registers a new bridge validator.
func (k Keeper) HandleRegisterBridgeValidator(ctx sdk.Context, msg types.MsgRegisterBridgeValidator) error {
	// Check if already registered
	if existing, found := k.GetBridgeValidator(ctx, msg.ValidatorAddress); found {
		// Update existing validator — preserve reputation and registration time
		existing.Active = true
		existing.PQCPubkey = msg.PQCPubkey
		existing.SupportedChains = msg.SupportedChains
		if err := k.SetBridgeValidator(ctx, existing); err != nil {
			return err
		}
	} else {
		validator := types.BridgeValidator{
			Address:         msg.ValidatorAddress,
			PQCPubkey:       msg.PQCPubkey,
			SupportedChains: msg.SupportedChains,
			Reputation:      1.0,
			Active:          true,
			RegisteredAt:    ctx.BlockHeight(),
		}
		if err := k.SetBridgeValidator(ctx, validator); err != nil {
			return err
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeValidatorRegistered,
		sdk.NewAttribute(types.AttributeKeyValidator, msg.ValidatorAddress),
	))

	k.Logger().Info("bridge validator registered",
		"validator", msg.ValidatorAddress,
		"chains", fmt.Sprintf("%v", msg.SupportedChains),
	)

	return nil
}

// HandleBridgeAttestation processes a bridge attestation from a validator.
func (k Keeper) HandleBridgeAttestation(ctx sdk.Context, msg types.MsgBridgeAttestation) error {
	// 1. Verify validator is registered and active
	validator, found := k.GetBridgeValidator(ctx, msg.Validator)
	if !found {
		return types.ErrValidatorNotRegistered.Wrapf("validator %s not registered", msg.Validator)
	}
	if !validator.Active {
		return types.ErrValidatorNotAuthorized.Wrapf("validator %s is not active", msg.Validator)
	}

	// 2. Verify validator supports this chain
	chainAuthorized := false
	for _, c := range validator.SupportedChains {
		if c == msg.Chain {
			chainAuthorized = true
			break
		}
	}
	if !chainAuthorized {
		return types.ErrValidatorNotAuthorized.Wrapf("validator %s not authorized for chain %s", msg.Validator, msg.Chain)
	}

	// 3. Verify PQC signature using Dilithium-5
	signBytes := msg.GetSignBytes()
	valid, err := k.pqcKeeper.PQCClient().DilithiumVerify(
		validator.PQCPubkey,
		signBytes,
		msg.PQCSignature,
	)
	if err != nil {
		return types.ErrInvalidPQCSignature.Wrapf("PQC verification error: %v", err)
	}
	if !valid {
		return types.ErrInvalidPQCSignature.Wrap("PQC signature verification failed")
	}

	// 4. Get the operation
	op, found := k.GetOperation(ctx, msg.OperationID)
	if !found {
		return types.ErrOperationNotFound.Wrapf("operation %s not found", msg.OperationID)
	}
	if op.Status == types.OpStatusExecuted || op.Status == types.OpStatusFailed {
		return types.ErrOperationAlreadyCompleted.Wrapf("operation %s already %s", msg.OperationID, op.Status)
	}

	// 5. Check for duplicate attestation
	for _, att := range op.Attestations {
		if att.Validator == msg.Validator {
			return types.ErrDuplicateAttestation.Wrapf("validator %s already attested to operation %s", msg.Validator, msg.OperationID)
		}
	}

	// 6. Add attestation
	attestation := types.Attestation{
		Validator:    msg.Validator,
		EventType:    msg.EventType,
		TxHash:       msg.TxHash,
		PQCSignature: msg.PQCSignature,
		Timestamp:    ctx.BlockHeight(),
	}
	op.Attestations = append(op.Attestations, attestation)

	// 7. Check if attestation threshold reached
	config := k.GetConfig(ctx)
	if len(op.Attestations) >= config.AttestationThreshold {
		// Check if challenge period has passed (for large transfers)
		if op.ChallengeEndTime != nil && ctx.BlockTime().Before(*op.ChallengeEndTime) {
			op.Status = types.OpStatusAttested
			k.Logger().Info("operation attested, waiting for challenge period",
				"operation_id", op.ID,
				"challenge_end", op.ChallengeEndTime,
			)
		} else {
			// Execute the operation
			if err := k.ExecuteOperation(ctx, &op); err != nil {
				op.Status = types.OpStatusFailed
				k.Logger().Error("operation execution failed", "operation_id", op.ID, "error", err)
			}
		}
	}

	if err := k.SetOperation(ctx, op); err != nil {
		return err
	}

	// 8. Emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeBridgeAttestation,
			sdk.NewAttribute(types.AttributeKeyOperationID, msg.OperationID),
			sdk.NewAttribute(types.AttributeKeyValidator, msg.Validator),
			sdk.NewAttribute(types.AttributeKeyChain, msg.Chain),
			sdk.NewAttribute(types.AttributeKeyPQCVerified, "true"),
			sdk.NewAttribute(types.AttributeKeyAttestations, fmt.Sprintf("%d", len(op.Attestations))),
		),
	})

	return nil
}

// ExecuteOperation executes a bridge operation after threshold is met.
func (k Keeper) ExecuteOperation(ctx sdk.Context, op *types.BridgeOperation) error {
	now := ctx.BlockTime()
	op.Status = types.OpStatusExecuted
	op.CompletedAt = &now

	if op.Type == types.OpTypeDeposit {
		// Update minted amounts
		locked := k.GetLockedAmount(ctx, op.SourceChain, op.Asset)
		currentMinted, _ := types.ParseAmount(locked.TotalMinted)
		amount, _ := types.ParseAmount(op.Amount)
		locked.TotalMinted = currentMinted.Add(amount).String()
		if err := k.SetLockedAmount(ctx, locked); err != nil {
			return err
		}
	} else if op.Type == types.OpTypeWithdrawal {
		// Reduce minted amounts
		locked := k.GetLockedAmount(ctx, op.DestChain, op.Asset)
		currentMinted, _ := types.ParseAmount(locked.TotalMinted)
		amount, _ := types.ParseAmount(op.Amount)
		if currentMinted.LT(amount) {
			return fmt.Errorf("minted balance insufficient for withdrawal: have %s, need %s", currentMinted, amount)
		}
		locked.TotalMinted = currentMinted.Sub(amount).String()
		currentLocked, _ := types.ParseAmount(locked.TotalLocked)
		if currentLocked.LT(amount) {
			return fmt.Errorf("locked balance insufficient for withdrawal: have %s, need %s", currentLocked, amount)
		}
		locked.TotalLocked = currentLocked.Sub(amount).String()
		if err := k.SetLockedAmount(ctx, locked); err != nil {
			return err
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeOperationExecuted,
		sdk.NewAttribute(types.AttributeKeyOperationID, op.ID),
		sdk.NewAttribute(types.AttributeKeyStatus, string(op.Status)),
	))

	k.Logger().Info("bridge operation executed",
		"operation_id", op.ID,
		"type", op.Type,
		"chain", op.SourceChain,
		"asset", op.Asset,
		"amount", op.Amount,
	)

	return nil
}

// CheckCircuitBreakerLimits validates transfer against circuit breaker limits.
func (k Keeper) CheckCircuitBreakerLimits(ctx sdk.Context, chain string, amount sdkmath.Int) error {
	cb := k.GetCircuitBreaker(ctx, chain)

	// Manual pause check
	if cb.Paused {
		return types.ErrBridgePaused.Wrapf("bridge for %s is paused: %s", chain, cb.PausedReason)
	}

	// Single transfer limit
	maxSingle, _ := types.ParseAmount(cb.MaxSingleTransfer)
	if !maxSingle.IsZero() && amount.GT(maxSingle) {
		return types.ErrExceedsSingleTransferLimit.Wrapf(
			"transfer %s exceeds single transfer limit %s for %s",
			amount.String(), maxSingle.String(), chain,
		)
	}

	// Daily limit — reset if we're past the reset window (approximately every 14400 blocks ~ 24h)
	const blocksPerDay int64 = 14400
	if ctx.BlockHeight()-cb.LastResetHeight > blocksPerDay {
		cb.CurrentDaily = "0"
		cb.LastResetHeight = ctx.BlockHeight()
		if err := k.SetCircuitBreaker(ctx, cb); err != nil {
			k.Logger().Error("failed to update circuit breaker", "error", err)
		}
	}

	dailyLimit, _ := types.ParseAmount(cb.DailyLimit)
	currentDaily, _ := types.ParseAmount(cb.CurrentDaily)
	if !dailyLimit.IsZero() && currentDaily.Add(amount).GT(dailyLimit) {
		return types.ErrExceedsDailyLimit.Wrapf(
			"transfer would exceed daily limit %s for %s (current daily: %s)",
			dailyLimit.String(), chain, currentDaily.String(),
		)
	}

	return nil
}

// IncrementDailyUsage updates the daily usage counter for a chain's circuit breaker.
func (k Keeper) IncrementDailyUsage(ctx sdk.Context, chain string, amount sdkmath.Int) {
	cb := k.GetCircuitBreaker(ctx, chain)
	currentDaily, _ := types.ParseAmount(cb.CurrentDaily)
	cb.CurrentDaily = currentDaily.Add(amount).String()
	if err := k.SetCircuitBreaker(ctx, cb); err != nil {
		k.Logger().Error("failed to update circuit breaker daily usage", "error", err)
	}
}
