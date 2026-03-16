//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	burnmod "github.com/qorechain/qorechain-core/x/burn"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// Keeper manages the rdk module state.
type Keeper struct {
	cdc              codec.Codec
	storeKey         storetypes.StoreKey
	burnKeeper       burnmod.BurnKeeper
	multilayerKeeper multilayermod.MultilayerKeeper
	rlKeeper         rlconsensusmod.RLConsensusKeeper
	bankKeeper       bankkeeper.BaseKeeper
	logger           log.Logger
}

// NewKeeper creates a new rdk keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	burnKeeper burnmod.BurnKeeper,
	multilayerKeeper multilayermod.MultilayerKeeper,
	rlKeeper rlconsensusmod.RLConsensusKeeper,
	bankKeeper bankkeeper.BaseKeeper,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		burnKeeper:       burnKeeper,
		multilayerKeeper: multilayerKeeper,
		rlKeeper:         rlKeeper,
		bankKeeper:       bankKeeper,
		logger:           logger.With("module", types.ModuleName),
	}
}

// Logger returns the module logger.
func (k Keeper) Logger() log.Logger { return k.logger }

// --- Rollup CRUD ---

func rollupKey(rollupID string) []byte {
	return append(types.RollupConfigPrefix, []byte(rollupID)...)
}

// GetRollup returns a rollup config by ID.
func (k Keeper) GetRollup(ctx sdk.Context, rollupID string) (*types.RollupConfig, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(rollupKey(rollupID))
	if bz == nil {
		return nil, types.ErrRollupNotFound
	}
	var config types.RollupConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rollup config: %w", err)
	}
	return &config, nil
}

// setRollup stores a rollup config.
func (k Keeper) setRollup(ctx sdk.Context, config types.RollupConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(config)
	if err != nil {
		return err
	}
	store.Set(rollupKey(config.RollupID), bz)
	return nil
}

// ListRollups returns all rollup configs.
func (k Keeper) ListRollups(ctx sdk.Context) ([]*types.RollupConfig, error) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.RollupConfigPrefix)
	defer iter.Close()

	var configs []*types.RollupConfig
	for ; iter.Valid(); iter.Next() {
		var config types.RollupConfig
		if err := json.Unmarshal(iter.Value(), &config); err != nil {
			continue
		}
		configs = append(configs, &config)
	}
	return configs, nil
}

// ListRollupsByCreator returns rollup configs filtered by creator.
func (k Keeper) ListRollupsByCreator(ctx sdk.Context, creator string) ([]*types.RollupConfig, error) {
	all, err := k.ListRollups(ctx)
	if err != nil {
		return nil, err
	}
	var filtered []*types.RollupConfig
	for _, c := range all {
		if c.Creator == creator {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

// --- Params ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		k.logger.Warn("failed to unmarshal rdk params, using defaults", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set rdk params: %v", err))
	}
	for _, rollup := range gs.Rollups {
		if err := k.setRollup(ctx, rollup); err != nil {
			panic(fmt.Sprintf("failed to set rollup config: %v", err))
		}
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	rollups, _ := k.ListRollups(ctx)
	var configs []types.RollupConfig
	for _, r := range rollups {
		configs = append(configs, *r)
	}
	return &types.GenesisState{
		Params:  k.GetParams(ctx),
		Rollups: configs,
		Batches: k.GetAllBatches(ctx),
	}
}

// GetAllBatches returns all settlement batches across all rollups.
func (k Keeper) GetAllBatches(ctx sdk.Context) []types.SettlementBatch {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.SettlementBatchPrefix)
	defer iter.Close()

	var batches []types.SettlementBatch
	for ; iter.Valid(); iter.Next() {
		var batch types.SettlementBatch
		if err := json.Unmarshal(iter.Value(), &batch); err != nil {
			continue
		}
		batches = append(batches, batch)
	}
	return batches
}

// rollupCount returns the count of all rollups.
func (k Keeper) rollupCount(ctx sdk.Context) uint32 {
	rollups, _ := k.ListRollups(ctx)
	return uint32(len(rollups))
}

// --- Batch helpers (used by settlement.go) ---

func batchKey(rollupID string, batchIndex uint64) []byte {
	return append(types.SettlementBatchPrefix, []byte(rollupID+"/"+strconv.FormatUint(batchIndex, 10))...)
}

func latestBatchKey(rollupID string) []byte {
	return append(types.LatestBatchPrefix, []byte(rollupID)...)
}

// GetBatch returns a settlement batch by rollup ID and batch index.
func (k Keeper) GetBatch(ctx sdk.Context, rollupID string, batchIndex uint64) (*types.SettlementBatch, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(batchKey(rollupID, batchIndex))
	if bz == nil {
		return nil, types.ErrBatchNotFound
	}
	var batch types.SettlementBatch
	if err := json.Unmarshal(bz, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch: %w", err)
	}
	return &batch, nil
}

// setBatch stores a settlement batch.
func (k Keeper) setBatch(ctx sdk.Context, batch types.SettlementBatch) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	store.Set(batchKey(batch.RollupID, batch.BatchIndex), bz)
	// Also update latest batch pointer
	store.Set(latestBatchKey(batch.RollupID), bz)
	return nil
}

// GetLatestBatch returns the latest settlement batch for a rollup.
func (k Keeper) GetLatestBatch(ctx sdk.Context, rollupID string) (*types.SettlementBatch, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(latestBatchKey(rollupID))
	if bz == nil {
		return nil, types.ErrBatchNotFound
	}
	var batch types.SettlementBatch
	if err := json.Unmarshal(bz, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal latest batch: %w", err)
	}
	return &batch, nil
}
