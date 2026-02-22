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

// RegisterSidechain creates a new sidechain layer in the QoreChain multi-layer architecture.
// Validates creator stake, checks maximum sidechain limit, assigns ICS chain ID, and stores config.
func (k Keeper) RegisterSidechain(ctx sdk.Context, msg *types.MsgRegisterSidechain) (*types.MsgRegisterSidechainResponse, error) {
	// Check if layer already exists
	if _, err := k.GetLayer(ctx, msg.LayerID); err == nil {
		return nil, types.ErrLayerAlreadyExists.Wrapf("layer %s already exists", msg.LayerID)
	}

	// Check max sidechains limit
	params := k.GetParams(ctx)
	activeSidechains := k.countLayersByTypeAndStatus(ctx, types.LayerTypeSidechain, types.LayerStatusActive)
	proposedSidechains := k.countLayersByTypeAndStatus(ctx, types.LayerTypeSidechain, types.LayerStatusProposed)
	if uint64(activeSidechains+proposedSidechains) >= params.MaxSidechains {
		return nil, types.ErrMaxSidechainsReached
	}

	// Generate ICS chain ID for the sidechain
	chainID := fmt.Sprintf("qorechain-%s", msg.LayerID)

	now := ctx.BlockTime()
	layer := types.LayerConfig{
		LayerID:                      msg.LayerID,
		LayerType:                    types.LayerTypeSidechain,
		Status:                       types.LayerStatusProposed,
		ChainID:                      chainID,
		Description:                  msg.Description,
		TargetBlockTimeMs:            msg.TargetBlockTimeMs,
		MaxTransactionsPerBlock:      msg.MaxTransactionsPerBlock,
		MaxGasPerBlock:               100000000, // Default 100M gas
		MinValidators:                msg.MinValidators,
		SettlementIntervalBlocks:     msg.SettlementIntervalBlocks,
		ChallengePeriodSeconds:       params.DefaultChallengePeriod,
		BaseFeeMultiplier:            "1.0", // Same as main chain by default
		CrossLayerFeeBundlingEnabled: params.CrossLayerFeeBundling,
		SupportedVMTypes:             msg.SupportedVMTypes,
		SupportedDomains:             msg.SupportedDomains,
		RegisteredAt:                 &now,
		Creator:                      msg.Creator,
	}

	if err := k.setLayer(ctx, layer); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSidechainRegistered,
		sdk.NewAttribute(types.AttributeKeyLayerID, msg.LayerID),
		sdk.NewAttribute(types.AttributeKeyLayerType, string(types.LayerTypeSidechain)),
		sdk.NewAttribute(types.AttributeKeyChainID, chainID),
		sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
		sdk.NewAttribute(types.AttributeKeyStatus, string(types.LayerStatusProposed)),
	))

	k.logger.Info("sidechain registered",
		"layer_id", msg.LayerID,
		"chain_id", chainID,
		"creator", msg.Creator,
	)

	return &types.MsgRegisterSidechainResponse{
		LayerID: msg.LayerID,
		ChainID: chainID,
		Status:  types.LayerStatusProposed,
	}, nil
}

// RegisterPaychain creates a new paychain layer for high-frequency microtransactions.
func (k Keeper) RegisterPaychain(ctx sdk.Context, msg *types.MsgRegisterPaychain) (*types.MsgRegisterPaychainResponse, error) {
	// Check if layer already exists
	if _, err := k.GetLayer(ctx, msg.LayerID); err == nil {
		return nil, types.ErrLayerAlreadyExists.Wrapf("layer %s already exists", msg.LayerID)
	}

	// Check max paychains limit
	params := k.GetParams(ctx)
	activePaychains := k.countLayersByTypeAndStatus(ctx, types.LayerTypePaychain, types.LayerStatusActive)
	proposedPaychains := k.countLayersByTypeAndStatus(ctx, types.LayerTypePaychain, types.LayerStatusProposed)
	if uint64(activePaychains+proposedPaychains) >= params.MaxPaychains {
		return nil, types.ErrMaxPaychainsReached
	}

	// Default fee multiplier for paychains (1/100th of main chain)
	feeMultiplier := msg.BaseFeeMultiplier
	if feeMultiplier == "" {
		feeMultiplier = "0.01"
	}

	now := ctx.BlockTime()
	layer := types.LayerConfig{
		LayerID:                      msg.LayerID,
		LayerType:                    types.LayerTypePaychain,
		Status:                       types.LayerStatusProposed,
		Description:                  msg.Description,
		TargetBlockTimeMs:            500, // Paychains target 500ms blocks
		MaxTransactionsPerBlock:      msg.MaxTransactionsPerBlock,
		MaxGasPerBlock:               50000000, // 50M gas for paychains
		MinValidators:                3,         // Paychains need fewer validators
		SettlementIntervalBlocks:     msg.SettlementIntervalBlocks,
		ChallengePeriodSeconds:       params.DefaultChallengePeriod,
		BaseFeeMultiplier:            feeMultiplier,
		CrossLayerFeeBundlingEnabled: params.CrossLayerFeeBundling,
		SupportedDomains:             []string{"payments", "microtx"},
		RegisteredAt:                 &now,
		Creator:                      msg.Creator,
	}

	if err := k.setLayer(ctx, layer); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypePaychainRegistered,
		sdk.NewAttribute(types.AttributeKeyLayerID, msg.LayerID),
		sdk.NewAttribute(types.AttributeKeyLayerType, string(types.LayerTypePaychain)),
		sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
		sdk.NewAttribute(types.AttributeKeyStatus, string(types.LayerStatusProposed)),
	))

	k.logger.Info("paychain registered",
		"layer_id", msg.LayerID,
		"creator", msg.Creator,
		"fee_multiplier", feeMultiplier,
	)

	return &types.MsgRegisterPaychainResponse{
		LayerID: msg.LayerID,
		Status:  types.LayerStatusProposed,
	}, nil
}

// UpdateLayerStatus changes a layer's status with valid transition enforcement.
// Valid transitions: PROPOSED->ACTIVE, PROPOSED->DECOMMISSIONED, ACTIVE->SUSPENDED,
// ACTIVE->DECOMMISSIONED, SUSPENDED->ACTIVE, SUSPENDED->DECOMMISSIONED.
func (k Keeper) UpdateLayerStatus(ctx sdk.Context, layerID string, newStatus types.LayerStatus, reason string) error {
	layer, err := k.GetLayer(ctx, layerID)
	if err != nil {
		return err
	}

	if !types.IsValidTransition(layer.Status, newStatus) {
		return types.ErrInvalidLayerTransition.Wrapf(
			"cannot transition layer %s from %s to %s",
			layerID, layer.Status, newStatus,
		)
	}

	layer.Status = newStatus

	if err := k.setLayer(ctx, *layer); err != nil {
		return err
	}

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeLayerStatusUpdated,
		sdk.NewAttribute(types.AttributeKeyLayerID, layerID),
		sdk.NewAttribute(types.AttributeKeyStatus, string(newStatus)),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
	))

	k.logger.Info("layer status updated",
		"layer_id", layerID,
		"new_status", string(newStatus),
		"reason", reason,
	)

	return nil
}

// ---- Layer Storage ----

// GetLayer retrieves a layer configuration from the KVStore.
func (k Keeper) GetLayer(ctx sdk.Context, layerID string) (*types.LayerConfig, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LayerKey(layerID))
	if bz == nil {
		return nil, types.ErrLayerNotFound.Wrapf("layer %s", layerID)
	}
	var layer types.LayerConfig
	if err := json.Unmarshal(bz, &layer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal layer %s: %w", layerID, err)
	}
	return &layer, nil
}

// setLayer stores a layer configuration in the KVStore.
func (k Keeper) setLayer(ctx sdk.Context, layer types.LayerConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(layer)
	if err != nil {
		return err
	}
	store.Set(types.LayerKey(layer.LayerID), bz)
	return nil
}

// GetAllLayers returns all registered layers.
func (k Keeper) GetAllLayers(ctx sdk.Context) ([]*types.LayerConfig, error) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.LayerKeyPrefix)
	defer iter.Close()

	var layers []*types.LayerConfig
	for ; iter.Valid(); iter.Next() {
		var layer types.LayerConfig
		if err := json.Unmarshal(iter.Value(), &layer); err != nil {
			continue
		}
		layers = append(layers, &layer)
	}
	return layers, nil
}

// GetLayersByType returns layers filtered by type.
func (k Keeper) GetLayersByType(ctx sdk.Context, layerType types.LayerType) ([]*types.LayerConfig, error) {
	all, err := k.GetAllLayers(ctx)
	if err != nil {
		return nil, err
	}
	var filtered []*types.LayerConfig
	for _, layer := range all {
		if layer.LayerType == layerType {
			filtered = append(filtered, layer)
		}
	}
	return filtered, nil
}

// countLayersByTypeAndStatus counts layers matching the given type and status.
func (k Keeper) countLayersByTypeAndStatus(ctx sdk.Context, layerType types.LayerType, status types.LayerStatus) int {
	all, err := k.GetAllLayers(ctx)
	if err != nil {
		return 0
	}
	count := 0
	for _, layer := range all {
		if layer.LayerType == layerType && layer.Status == status {
			count++
		}
	}
	return count
}

// getActiveLayers returns all layers with active status (used by QCAI router).
func (k Keeper) getActiveLayers(ctx sdk.Context) []*types.LayerConfig {
	all, _ := k.GetAllLayers(ctx)
	var active []*types.LayerConfig
	for _, layer := range all {
		if layer.Status == types.LayerStatusActive {
			active = append(active, layer)
		}
	}
	return active
}

// updateLastAnchorTime updates the last anchor timestamp for a layer.
func (k Keeper) updateLastAnchorTime(ctx sdk.Context, layerID string, t time.Time) error {
	layer, err := k.GetLayer(ctx, layerID)
	if err != nil {
		return err
	}
	layer.LastAnchorAt = &t
	return k.setLayer(ctx, *layer)
}
