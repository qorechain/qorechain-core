package keeper

import (
	"encoding/json"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
	"github.com/qorechain/qorechain-core/x/qca/types"
)

// StakingReader provides read-only staking data for pool classification and bonding curve.
type StakingReader interface {
	// GetValidatorTokens returns total delegated tokens for a validator (in uqor).
	GetValidatorTokens(ctx sdk.Context, valAddr string) uint64
}

// RLReader provides read-only access to RL consensus parameters.
type RLReader interface {
	// GetDynamicPoolWeights returns the RL-adjusted pool weights (rpos, dpos).
	// Returns empty strings if RL is not active.
	GetDynamicPoolWeights(ctx sdk.Context) (rpos string, dpos string)
}

// Keeper manages the x/qca module state.
type Keeper struct {
	cdc              codec.Codec
	storeKey         storetypes.StoreKey
	reputationKeeper reputationkeeper.Keeper
	selector         types.ValidatorSelector
	logger           log.Logger
	stakingKeeper    StakingReader // optional, set after creation
	rlKeeper         RLReader      // optional, set after creation
}

// NewKeeper creates a new QCA keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	reputationKeeper reputationkeeper.Keeper,
	selector types.ValidatorSelector,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		reputationKeeper: reputationKeeper,
		selector:         selector,
		logger:           logger.With("module", types.ModuleName),
	}
}

func (k Keeper) Logger() log.Logger { return k.logger }

// SetStakingKeeper sets the optional staking keeper dependency.
func (k *Keeper) SetStakingKeeper(sk StakingReader) {
	k.stakingKeeper = sk
}

// SetRLKeeper sets the optional RL consensus keeper dependency.
func (k *Keeper) SetRLKeeper(rl RLReader) {
	k.rlKeeper = rl
}

// ---- Config ----

func (k Keeper) GetConfig(ctx sdk.Context) types.QCAConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConfigKey)
	if bz == nil {
		return types.DefaultQCAConfig()
	}
	var cfg types.QCAConfig
	if err := json.Unmarshal(bz, &cfg); err != nil {
		return types.DefaultQCAConfig()
	}
	return cfg
}

func (k Keeper) SetConfig(ctx sdk.Context, cfg types.QCAConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	store.Set(types.ConfigKey, bz)
	return nil
}

// ---- Stats ----

func (k Keeper) GetStats(ctx sdk.Context) types.QCAStats {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.StatsKey)
	if bz == nil {
		return types.QCAStats{}
	}
	var stats types.QCAStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.QCAStats{}
	}
	return stats
}

func (k Keeper) SetStats(ctx sdk.Context, stats types.QCAStats) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(stats)
	store.Set(types.StatsKey, bz)
}

// GetReputationWeightedProposer selects the next block proposer using reputation weighting.
func (k Keeper) GetReputationWeightedProposer(
	ctx sdk.Context,
	validators []types.ValidatorInfo,
) string {
	config := k.GetConfig(ctx)

	// Get reputation scores for all validators
	scores := make(map[string]float64)
	for _, val := range validators {
		scores[val.Address] = k.reputationKeeper.CalculateReputation(ctx, val.Address)
	}

	var selected string
	if config.UseReputationWeighting {
		blockHash := ctx.BlockHeader().LastBlockId.Hash
		addr, err := k.selector.SelectProposer(validators, scores, blockHash, ctx.BlockHeight())
		if err != nil {
			k.logger.Warn("proposer selection failed", "error", err)
			return ""
		}
		selected = addr

		stats := k.GetStats(ctx)
		stats.ProposerSelections++
		stats.ReputationWeighted++
		k.SetStats(ctx, stats)
	}

	return selected
}

// ---- Pool & Slashing KV helpers (shared between public and proprietary builds) ----

// setPoolClassificationKV stores a pool classification in the KV store.
func (k Keeper) setPoolClassificationKV(ctx sdk.Context, pc types.PoolClassification) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(pc)
	store.Set(types.PoolClassificationKey(pc.ValidatorAddr), bz)
}

// setSlashingRecordKV stores a slashing record in the KV store.
func (k Keeper) setSlashingRecordKV(ctx sdk.Context, record types.SlashingRecord) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(record)
	store.Set(types.SlashingRecordKey(record.ValidatorAddr, record.InfractionHeight), bz)
}

// getAllPoolClassifications returns all stored pool classifications.
func (k Keeper) getAllPoolClassifications(ctx sdk.Context) []types.PoolClassification {
	store := ctx.KVStore(k.storeKey)
	var result []types.PoolClassification
	iter := store.Iterator(types.PoolClassificationPrefix, storetypes.PrefixEndBytes(types.PoolClassificationPrefix))
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var pc types.PoolClassification
		if err := json.Unmarshal(iter.Value(), &pc); err != nil {
			continue
		}
		result = append(result, pc)
	}
	return result
}

// getAllSlashingRecords returns all stored slashing records.
func (k Keeper) getAllSlashingRecords(ctx sdk.Context) []types.SlashingRecord {
	store := ctx.KVStore(k.storeKey)
	var result []types.SlashingRecord
	iter := store.Iterator(types.SlashingRecordPrefix, storetypes.PrefixEndBytes(types.SlashingRecordPrefix))
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var record types.SlashingRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		result = append(result, record)
	}
	return result
}

// ---- Genesis ----

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetConfig(ctx, gs.Config); err != nil {
		panic(err)
	}
	k.SetStats(ctx, gs.Stats)
	for _, pc := range gs.PoolClassifications {
		k.setPoolClassificationKV(ctx, pc)
	}
	for _, record := range gs.SlashingRecords {
		k.setSlashingRecordKV(ctx, record)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Config:              k.GetConfig(ctx),
		Stats:               k.GetStats(ctx),
		PoolClassifications: k.getAllPoolClassifications(ctx),
		SlashingRecords:     k.getAllSlashingRecords(ctx),
	}
}
