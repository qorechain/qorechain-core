//go:build proprietary

package keeper

import (
	"context"
	"encoding/json"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// Keeper manages the x/ai module state and provides AI operations.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	engine   types.AIEngine
	logger   log.Logger
}

// NewKeeper creates a new AI keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	engine types.AIEngine,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		engine:   engine,
		logger:   logger.With("module", types.ModuleName),
	}
}

// Engine returns the AI engine.
func (k Keeper) Engine() types.AIEngine {
	return k.engine
}

// Logger returns the module logger.
func (k Keeper) Logger() log.Logger {
	return k.logger
}

// ---- Config ----

func (k Keeper) GetConfig(ctx sdk.Context) types.AIConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConfigKey)
	if bz == nil {
		return types.DefaultAIConfig()
	}
	var cfg types.AIConfig
	if err := json.Unmarshal(bz, &cfg); err != nil {
		return types.DefaultAIConfig()
	}
	return cfg
}

func (k Keeper) SetConfig(ctx sdk.Context, cfg types.AIConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	store.Set(types.ConfigKey, bz)
	return nil
}

// ---- Stats ----

func (k Keeper) GetStats(ctx sdk.Context) types.AIStats {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.StatsKey)
	if bz == nil {
		return types.AIStats{}
	}
	var stats types.AIStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.AIStats{}
	}
	return stats
}

func (k Keeper) SetStats(ctx sdk.Context, stats types.AIStats) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(stats)
	store.Set(types.StatsKey, bz)
}

func (k Keeper) IncrementTxsRouted(ctx sdk.Context) {
	stats := k.GetStats(ctx)
	stats.TxsRouted++
	k.SetStats(ctx, stats)
}

func (k Keeper) IncrementAnomaliesDetected(ctx sdk.Context) {
	stats := k.GetStats(ctx)
	stats.AnomaliesDetected++
	k.SetStats(ctx, stats)
}

// ---- Flagged TXs ----

func (k Keeper) FlagTransaction(ctx sdk.Context, flagged types.FlaggedTx) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.FlaggedTxPrefix, []byte(flagged.TxHash)...)
	bz, _ := json.Marshal(flagged)
	store.Set(key, bz)
}

// ---- AI Operations ----

// AnalyzeTransaction runs routing and anomaly detection on a transaction.
func (k Keeper) AnalyzeTransaction(ctx sdk.Context, tx types.TransactionInfo, history []types.TransactionInfo) (*types.AnomalyResult, error) {
	goCtx := context.Background()

	// Run anomaly detection
	result, err := k.engine.DetectAnomaly(goCtx, tx, history)
	if err != nil {
		return nil, err
	}

	k.IncrementTxsRouted(ctx)

	if result.IsAnomalous {
		k.IncrementAnomaliesDetected(ctx)
		k.FlagTransaction(ctx, types.FlaggedTx{
			TxHash:       tx.TxHash,
			AnomalyScore: result.Score,
			Flags:        result.Flags,
			Height:       ctx.BlockHeight(),
		})
	}

	return result, nil
}

// ---- Genesis ----

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetConfig(ctx, gs.Config); err != nil {
		panic(err)
	}
	k.SetStats(ctx, gs.Stats)
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Config: k.GetConfig(ctx),
		Stats:  k.GetStats(ctx),
	}
}
