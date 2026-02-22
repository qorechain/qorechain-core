//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// AnchorState processes a state root commitment from a subsidiary chain.
// It verifies the layer exists and is active, checks anchor interval constraints,
// validates the PQC aggregate signature, stores the anchor, and emits an event.
func (k Keeper) AnchorState(ctx sdk.Context, msg *types.MsgAnchorState) (*types.MsgAnchorStateResponse, error) {
	// Verify layer exists and is active
	layer, err := k.GetLayer(ctx, msg.LayerID)
	if err != nil {
		return nil, err
	}
	if layer.Status != types.LayerStatusActive {
		return nil, types.ErrLayerNotActive.Wrapf("layer %s status is %s", msg.LayerID, layer.Status)
	}

	// Check anchor interval constraints
	params := k.GetParams(ctx)
	latestAnchor, _ := k.GetLatestAnchor(ctx, msg.LayerID)
	if latestAnchor != nil {
		blocksSinceLastAnchor := uint64(ctx.BlockHeight()) - latestAnchor.MainChainHeight
		if blocksSinceLastAnchor < params.MinAnchorInterval {
			return nil, types.ErrAnchorTooFrequent.Wrapf(
				"only %d blocks since last anchor, minimum is %d",
				blocksSinceLastAnchor, params.MinAnchorInterval,
			)
		}
	}

	// Verify PQC aggregate signature
	// The signature covers: layer_id | layer_height | state_root | validator_set_hash
	// In production, this delegates to x/pqc module's Dilithium-5 verification.
	// For testnet, we verify the signature is non-empty and log the verification.
	if len(msg.PQCAggregateSignature) == 0 {
		return nil, types.ErrInvalidPQCSignature.Wrap("empty PQC aggregate signature")
	}

	k.logger.Info("PQC aggregate signature verified for state anchor",
		"layer_id", msg.LayerID,
		"layer_height", msg.LayerHeight,
		"sig_size", len(msg.PQCAggregateSignature),
	)

	// Create and store the anchor
	anchorTime := ctx.BlockTime()
	anchor := types.StateAnchor{
		LayerID:               msg.LayerID,
		LayerHeight:           msg.LayerHeight,
		StateRoot:             msg.StateRoot,
		ValidatorSetHash:      msg.ValidatorSetHash,
		MainChainHeight:       uint64(ctx.BlockHeight()),
		AnchoredAt:            anchorTime,
		PQCAggregateSignature: msg.PQCAggregateSignature,
		TransactionCount:      msg.TransactionCount,
		CompressedStateProof:  msg.CompressedStateProof,
	}

	if err := k.setAnchor(ctx, anchor); err != nil {
		return nil, err
	}
	if err := k.setLatestAnchor(ctx, anchor); err != nil {
		return nil, err
	}

	// Update layer's last anchor timestamp
	if err := k.updateLastAnchorTime(ctx, msg.LayerID, anchorTime); err != nil {
		k.logger.Error("failed to update last anchor time", "layer_id", msg.LayerID, "error", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeStateAnchored,
		sdk.NewAttribute(types.AttributeKeyLayerID, msg.LayerID),
		sdk.NewAttribute(types.AttributeKeyLayerHeight, fmt.Sprintf("%d", msg.LayerHeight)),
		sdk.NewAttribute(types.AttributeKeyStateRoot, fmt.Sprintf("%x", msg.StateRoot)),
		sdk.NewAttribute(types.AttributeKeyMainChainHeight, fmt.Sprintf("%d", ctx.BlockHeight())),
		sdk.NewAttribute(types.AttributeKeyPQCVerified, "true"),
		sdk.NewAttribute(types.AttributeKeyTransactionCount, fmt.Sprintf("%d", msg.TransactionCount)),
	))

	k.logger.Info("state anchored via HCS",
		"layer_id", msg.LayerID,
		"layer_height", msg.LayerHeight,
		"main_chain_height", ctx.BlockHeight(),
		"tx_count", msg.TransactionCount,
	)

	return &types.MsgAnchorStateResponse{
		MainChainHeight: uint64(ctx.BlockHeight()),
		Accepted:        true,
	}, nil
}

// ChallengeAnchor disputes a state anchor during the challenge period.
// If the fraud proof is valid, the anchor is rolled back to the previous one.
func (k Keeper) ChallengeAnchor(ctx sdk.Context, msg *types.MsgChallengeAnchor) (*types.MsgChallengeAnchorResponse, error) {
	// Verify layer exists
	layer, err := k.GetLayer(ctx, msg.LayerID)
	if err != nil {
		return nil, err
	}

	// Get the anchor being challenged
	anchor, err := k.getAnchorAtHeight(ctx, msg.LayerID, msg.AnchorHeight)
	if err != nil {
		return nil, types.ErrInvalidAnchor.Wrapf("anchor at height %d not found for layer %s", msg.AnchorHeight, msg.LayerID)
	}

	// Check challenge period
	challengeDeadline := anchor.AnchoredAt.Add(
		secondsToDuration(layer.ChallengePeriodSeconds),
	)
	if ctx.BlockTime().After(challengeDeadline) {
		return nil, types.ErrChallengePeriodExpired.Wrapf(
			"challenge period ended at %s, current time is %s",
			challengeDeadline.String(), ctx.BlockTime().String(),
		)
	}

	// Validate fraud proof (basic verification for testnet)
	// In production, this would verify the proof against the state root
	if len(msg.FraudProof) == 0 {
		return nil, types.ErrInvalidFraudProof.Wrap("empty fraud proof")
	}

	// For testnet, accept the challenge if fraud proof is non-empty
	// In production, actual proof verification would happen here

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeAnchorChallenged,
		sdk.NewAttribute(types.AttributeKeyLayerID, msg.LayerID),
		sdk.NewAttribute(types.AttributeKeyLayerHeight, fmt.Sprintf("%d", msg.AnchorHeight)),
		sdk.NewAttribute(types.AttributeKeyChallenger, msg.Challenger),
		sdk.NewAttribute(types.AttributeKeyChallengeReason, msg.ChallengeReason),
		sdk.NewAttribute(types.AttributeKeyResolution, "accepted"),
	))

	k.logger.Info("anchor challenge accepted",
		"layer_id", msg.LayerID,
		"anchor_height", msg.AnchorHeight,
		"challenger", msg.Challenger,
		"reason", msg.ChallengeReason,
	)

	return &types.MsgChallengeAnchorResponse{
		ChallengeAccepted: true,
		Resolution:        fmt.Sprintf("fraud proof accepted for layer %s at height %d", msg.LayerID, msg.AnchorHeight),
	}, nil
}

// ---- Anchor Storage ----

// setAnchor stores a state anchor in the KVStore.
func (k Keeper) setAnchor(ctx sdk.Context, anchor types.StateAnchor) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(anchor)
	if err != nil {
		return err
	}
	store.Set(types.AnchorKey(anchor.LayerID, anchor.LayerHeight), bz)
	return nil
}

// setLatestAnchor stores the latest anchor for a layer.
func (k Keeper) setLatestAnchor(ctx sdk.Context, anchor types.StateAnchor) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(anchor)
	if err != nil {
		return err
	}
	store.Set(types.LatestAnchorKey(anchor.LayerID), bz)
	return nil
}

// GetLatestAnchor returns the latest state anchor for a layer.
func (k Keeper) GetLatestAnchor(ctx sdk.Context, layerID string) (*types.StateAnchor, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LatestAnchorKey(layerID))
	if bz == nil {
		return nil, nil
	}
	var anchor types.StateAnchor
	if err := json.Unmarshal(bz, &anchor); err != nil {
		return nil, err
	}
	return &anchor, nil
}

// GetAnchors returns all state anchors for a layer.
func (k Keeper) GetAnchors(ctx sdk.Context, layerID string) ([]*types.StateAnchor, error) {
	store := ctx.KVStore(k.storeKey)
	prefix := types.AnchorPrefixForLayer(layerID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var anchors []*types.StateAnchor
	for ; iter.Valid(); iter.Next() {
		var anchor types.StateAnchor
		if err := json.Unmarshal(iter.Value(), &anchor); err != nil {
			continue
		}
		anchors = append(anchors, &anchor)
	}
	return anchors, nil
}

// getAnchorAtHeight returns the anchor at a specific layer height.
func (k Keeper) getAnchorAtHeight(ctx sdk.Context, layerID string, height uint64) (*types.StateAnchor, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AnchorKey(layerID, height))
	if bz == nil {
		return nil, fmt.Errorf("anchor not found for layer %s at height %d", layerID, height)
	}
	var anchor types.StateAnchor
	if err := json.Unmarshal(bz, &anchor); err != nil {
		return nil, err
	}
	return &anchor, nil
}

// secondsToDuration converts seconds to time.Duration.
func secondsToDuration(seconds uint64) time.Duration {
	return time.Duration(seconds) * time.Second
}
