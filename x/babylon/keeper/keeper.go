//go:build proprietary

package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/babylon/types"
)

// Keeper manages the babylon module state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	logger   log.Logger
}

// NewKeeper creates a new babylon keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		logger:   logger.With("module", types.ModuleName),
	}
}

// Logger returns the module logger.
func (k Keeper) Logger() log.Logger { return k.logger }

// --- Config ---

// GetConfig returns the BTC restaking configuration.
func (k Keeper) GetConfig(ctx sdk.Context) types.BTCRestakingConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConfigKey)
	if bz == nil {
		return types.DefaultBTCRestakingConfig()
	}
	var config types.BTCRestakingConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		k.logger.Warn("failed to unmarshal babylon config, using defaults", "error", err)
		return types.DefaultBTCRestakingConfig()
	}
	return config
}

// SetConfig stores the BTC restaking configuration.
func (k Keeper) SetConfig(ctx sdk.Context, config types.BTCRestakingConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(config)
	if err != nil {
		return err
	}
	store.Set(types.ConfigKey, bz)
	return nil
}

// IsEnabled returns whether BTC restaking is enabled.
func (k Keeper) IsEnabled(ctx sdk.Context) bool {
	return k.GetConfig(ctx).Enabled
}

// --- Staking Positions ---

func positionKey(id string) []byte {
	return append(types.StakingPositionPrefix, []byte(id)...)
}

// GetStakingPosition returns a staking position by ID.
func (k Keeper) GetStakingPosition(ctx sdk.Context, id string) (types.BTCStakingPosition, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(positionKey(id))
	if bz == nil {
		return types.BTCStakingPosition{}, false
	}
	var pos types.BTCStakingPosition
	if err := json.Unmarshal(bz, &pos); err != nil {
		return types.BTCStakingPosition{}, false
	}
	return pos, true
}

// SetStakingPosition stores a staking position.
func (k Keeper) SetStakingPosition(ctx sdk.Context, pos types.BTCStakingPosition) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(pos)
	if err != nil {
		return err
	}
	store.Set(positionKey(pos.ID), bz)
	return nil
}

// GetAllPositions returns all staking positions.
func (k Keeper) GetAllPositions(ctx sdk.Context) []types.BTCStakingPosition {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.StakingPositionPrefix)
	defer iter.Close()

	var positions []types.BTCStakingPosition
	for ; iter.Valid(); iter.Next() {
		var pos types.BTCStakingPosition
		if err := json.Unmarshal(iter.Value(), &pos); err != nil {
			continue
		}
		positions = append(positions, pos)
	}
	return positions
}

// --- Checkpoints ---

func checkpointKey(epoch uint64) []byte {
	key := make([]byte, len(types.CheckpointPrefix)+8)
	copy(key, types.CheckpointPrefix)
	binary.BigEndian.PutUint64(key[len(types.CheckpointPrefix):], epoch)
	return key
}

// GetCheckpoint returns a checkpoint by epoch number.
func (k Keeper) GetCheckpoint(ctx sdk.Context, epoch uint64) (types.BTCCheckpoint, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(checkpointKey(epoch))
	if bz == nil {
		return types.BTCCheckpoint{}, false
	}
	var cp types.BTCCheckpoint
	if err := json.Unmarshal(bz, &cp); err != nil {
		return types.BTCCheckpoint{}, false
	}
	return cp, true
}

// SetCheckpoint stores a checkpoint.
func (k Keeper) SetCheckpoint(ctx sdk.Context, cp types.BTCCheckpoint) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	store.Set(checkpointKey(cp.EpochNum), bz)
	return nil
}

// --- Epochs ---

var currentEpochKey = []byte("babylon/current_epoch")

// GetCurrentEpoch returns the current epoch number.
func (k Keeper) GetCurrentEpoch(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(currentEpochKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func epochSnapshotKey(epoch uint64) []byte {
	key := make([]byte, len(types.EpochSnapshotPrefix)+8)
	copy(key, types.EpochSnapshotPrefix)
	binary.BigEndian.PutUint64(key[len(types.EpochSnapshotPrefix):], epoch)
	return key
}

// GetEpochSnapshot returns the snapshot for a given epoch.
func (k Keeper) GetEpochSnapshot(ctx sdk.Context, epoch uint64) (types.BabylonEpochSnapshot, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(epochSnapshotKey(epoch))
	if bz == nil {
		return types.BabylonEpochSnapshot{}, false
	}
	var snap types.BabylonEpochSnapshot
	if err := json.Unmarshal(bz, &snap); err != nil {
		return types.BabylonEpochSnapshot{}, false
	}
	return snap, true
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetConfig(ctx, gs.Config); err != nil {
		panic(fmt.Sprintf("failed to set babylon config: %v", err))
	}
	for _, pos := range gs.Positions {
		if err := k.SetStakingPosition(ctx, pos); err != nil {
			panic(fmt.Sprintf("failed to set staking position: %v", err))
		}
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Config:    k.GetConfig(ctx),
		Positions: k.GetAllPositions(ctx),
	}
}
