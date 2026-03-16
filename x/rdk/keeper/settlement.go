//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"
	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// SubmitBatch validates and stores a settlement batch, anchoring state to x/multilayer.
func (k Keeper) SubmitBatch(ctx sdk.Context, batch types.SettlementBatch) error {
	// Validate rollup is active
	rollup, err := k.GetRollup(ctx, batch.RollupID)
	if err != nil {
		return err
	}
	if rollup.Status != types.RollupStatusActive {
		return types.ErrRollupNotActive
	}

	// Validate proof matches settlement mode
	switch rollup.SettlementMode {
	case types.SettlementZK:
		if len(batch.Proof) == 0 {
			return types.ErrProofRequired
		}
		batch.ProofType = rollup.ProofConfig.System
	case types.SettlementOptimistic:
		batch.ProofType = types.ProofSystemFraud
	case types.SettlementBased:
		batch.ProofType = types.ProofSystemNone
		batch.SequencerMode = types.SequencerBased
	case types.SettlementSovereign:
		batch.ProofType = types.ProofSystemNone
	}

	// Set metadata
	batch.SubmittedAt = ctx.BlockHeight()
	batch.Status = types.BatchSubmitted

	// Store batch
	if err := k.setBatch(ctx, batch); err != nil {
		return err
	}

	// Anchor state to x/multilayer (non-fatal)
	anchorMsg := &multilayertypes.MsgAnchorState{
		Relayer:          rollup.Creator,
		LayerID:          rollup.LayerID,
		LayerHeight:      batch.BatchIndex,
		StateRoot:        batch.StateRoot,
		TransactionCount: batch.TxCount,
	}
	if _, anchorErr := k.multilayerKeeper.AnchorState(ctx, anchorMsg); anchorErr != nil {
		k.logger.Warn("state anchoring failed (non-fatal)", "error", anchorErr, "rollup_id", batch.RollupID)
	}

	// For ZK: if proof present and non-empty, auto-finalize (stub verification: accept any non-empty proof)
	if rollup.SettlementMode == types.SettlementZK && len(batch.Proof) > 0 {
		batch.Status = types.BatchFinalized
		batch.FinalizedAt = ctx.BlockHeight()
		if err := k.setBatch(ctx, batch); err != nil {
			return err
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventBatchFinalized,
			sdk.NewAttribute("rollup_id", batch.RollupID),
			sdk.NewAttribute("batch_index", strconv.FormatUint(batch.BatchIndex, 10)),
			sdk.NewAttribute("proof_type", string(batch.ProofType)),
		))

		k.logger.Info("ZK batch auto-finalized", "rollup_id", batch.RollupID, "batch", batch.BatchIndex)
		return nil
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventBatchSubmitted,
		sdk.NewAttribute("rollup_id", batch.RollupID),
		sdk.NewAttribute("batch_index", strconv.FormatUint(batch.BatchIndex, 10)),
		sdk.NewAttribute("tx_count", strconv.FormatUint(batch.TxCount, 10)),
	))

	k.logger.Info("batch submitted", "rollup_id", batch.RollupID, "batch", batch.BatchIndex, "mode", rollup.SettlementMode)
	return nil
}

// ChallengeBatch submits a fraud proof challenge against a batch. Only valid for optimistic rollups.
func (k Keeper) ChallengeBatch(ctx sdk.Context, rollupID string, batchIndex uint64, proof []byte) error {
	rollup, err := k.GetRollup(ctx, rollupID)
	if err != nil {
		return err
	}

	// Only optimistic rollups support challenges
	if rollup.SettlementMode != types.SettlementOptimistic {
		return types.ErrChallengeWindowClosed
	}

	batch, err := k.GetBatch(ctx, rollupID, batchIndex)
	if err != nil {
		return err
	}

	// Must be in submitted state
	if batch.Status != types.BatchSubmitted {
		return types.ErrBatchAlreadyFinalized
	}

	// Check challenge window
	windowBlocks := int64(rollup.ProofConfig.ChallengeWindowSec / 6) // ~6s per block
	if ctx.BlockHeight()-batch.SubmittedAt > windowBlocks {
		return types.ErrChallengeWindowClosed
	}

	// Validate proof is provided
	if len(proof) == 0 {
		return types.ErrInvalidProof
	}

	// Mark as challenged
	batch.Status = types.BatchChallenged
	if err := k.setBatch(ctx, *batch); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventBatchChallenged,
		sdk.NewAttribute("rollup_id", rollupID),
		sdk.NewAttribute("batch_index", strconv.FormatUint(batchIndex, 10)),
	))

	k.logger.Info("batch challenged", "rollup_id", rollupID, "batch", batchIndex)
	return nil
}

// FinalizeBatch manually finalizes a batch. For optimistic: challenge window must have expired.
func (k Keeper) FinalizeBatch(ctx sdk.Context, rollupID string, batchIndex uint64) error {
	rollup, err := k.GetRollup(ctx, rollupID)
	if err != nil {
		return err
	}

	batch, err := k.GetBatch(ctx, rollupID, batchIndex)
	if err != nil {
		return err
	}

	if batch.Status == types.BatchFinalized {
		return types.ErrBatchAlreadyFinalized
	}

	if batch.Status == types.BatchChallenged {
		return fmt.Errorf("batch is challenged and cannot be finalized")
	}

	// For optimistic: check challenge window has expired
	if rollup.SettlementMode == types.SettlementOptimistic {
		windowBlocks := int64(rollup.ProofConfig.ChallengeWindowSec / 6)
		if ctx.BlockHeight()-batch.SubmittedAt < windowBlocks {
			return fmt.Errorf("challenge window has not expired yet")
		}
	}

	batch.Status = types.BatchFinalized
	batch.FinalizedAt = ctx.BlockHeight()
	if err := k.setBatch(ctx, *batch); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventBatchFinalized,
		sdk.NewAttribute("rollup_id", rollupID),
		sdk.NewAttribute("batch_index", strconv.FormatUint(batchIndex, 10)),
	))

	k.logger.Info("batch finalized", "rollup_id", rollupID, "batch", batchIndex)
	return nil
}

// EndBlockSettlement runs in the EndBlocker to auto-finalize batches and prune blobs.
func (k Keeper) EndBlockSettlement(ctx sdk.Context) error {
	rollups, err := k.ListRollups(ctx)
	if err != nil {
		return err
	}

	for _, rollup := range rollups {
		if rollup.Status != types.RollupStatusActive {
			continue
		}

		switch rollup.SettlementMode {
		case types.SettlementOptimistic:
			k.autoFinalizeOptimistic(ctx, rollup)
		case types.SettlementBased:
			k.autoFinalizeBased(ctx, rollup)
		}
	}

	// Prune expired DA blobs
	if _, err := k.PruneExpiredBlobs(ctx); err != nil {
		k.logger.Warn("blob pruning error (non-fatal)", "error", err)
	}

	return nil
}

// autoFinalizeOptimistic auto-finalizes all pending optimistic batches past the challenge window.
func (k Keeper) autoFinalizeOptimistic(ctx sdk.Context, rollup *types.RollupConfig) {
	windowBlocks := int64(rollup.ProofConfig.ChallengeWindowSec / 6)

	store := ctx.KVStore(k.storeKey)
	prefix := append(types.SettlementBatchPrefix, []byte(rollup.RollupID+"/")...)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var batch types.SettlementBatch
		if err := json.Unmarshal(iter.Value(), &batch); err != nil {
			continue
		}
		if batch.Status == types.BatchSubmitted &&
			ctx.BlockHeight()-batch.SubmittedAt >= windowBlocks {
			batch.Status = types.BatchFinalized
			batch.FinalizedAt = ctx.BlockHeight()
			bz, err := json.Marshal(batch)
			if err != nil {
				continue
			}
			store.Set(iter.Key(), bz)
			// Also update latest batch pointer if this is the latest
			store.Set(latestBatchKey(rollup.RollupID), bz)

			ctx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventBatchFinalized,
				sdk.NewAttribute("rollup_id", rollup.RollupID),
				sdk.NewAttribute("batch_index", strconv.FormatUint(batch.BatchIndex, 10)),
				sdk.NewAttribute("auto", "true"),
			))

			k.logger.Info("auto-finalized optimistic batch", "rollup_id", rollup.RollupID, "batch", batch.BatchIndex)
		}
	}
}

// autoFinalizeBased auto-finalizes based rollup batches (L1 finality = rollup finality).
func (k Keeper) autoFinalizeBased(ctx sdk.Context, rollup *types.RollupConfig) {
	latestBatch, err := k.GetLatestBatch(ctx, rollup.RollupID)
	if err != nil {
		return
	}

	// For based rollups, finalize after a short confirmation delay (use current block as L1 finality proxy)
	if latestBatch.Status == types.BatchSubmitted &&
		ctx.BlockHeight()-latestBatch.SubmittedAt >= 2 {
		latestBatch.Status = types.BatchFinalized
		latestBatch.FinalizedAt = ctx.BlockHeight()
		if err := k.setBatch(ctx, *latestBatch); err != nil {
			k.logger.Warn("auto-finalize based failed", "error", err, "rollup_id", rollup.RollupID)
			return
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventBatchFinalized,
			sdk.NewAttribute("rollup_id", rollup.RollupID),
			sdk.NewAttribute("batch_index", strconv.FormatUint(latestBatch.BatchIndex, 10)),
			sdk.NewAttribute("auto", "true"),
			sdk.NewAttribute("settlement", "based"),
		))

		k.logger.Info("auto-finalized based batch", "rollup_id", rollup.RollupID, "batch", latestBatch.BatchIndex)
	}
}
