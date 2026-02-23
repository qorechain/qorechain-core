//go:build proprietary

package keeper

import (
	"encoding/json"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/ffi"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// Keeper manages the x/pqc module state and provides PQC operations.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	pqc      ffi.PQCClient
	logger   log.Logger
}

// NewKeeper creates a new PQC keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	pqcClient ffi.PQCClient,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		pqc:      pqcClient,
		logger:   logger.With("module", types.ModuleName),
	}
}

// PQCClient returns the underlying PQC FFI client.
func (k Keeper) PQCClient() ffi.PQCClient {
	return k.pqc
}

// Logger returns the module logger.
func (k Keeper) Logger() log.Logger {
	return k.logger
}

// ---- Params ----

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// ---- Accounts ----

func accountKey(address string) []byte {
	return append(types.AccountPrefix, []byte(address)...)
}

// GetPQCAccount returns the PQC account info for an address.
func (k Keeper) GetPQCAccount(ctx sdk.Context, address string) (types.PQCAccountInfo, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(accountKey(address))
	if bz == nil {
		return types.PQCAccountInfo{}, false
	}
	var info types.PQCAccountInfo
	if err := json.Unmarshal(bz, &info); err != nil {
		return types.PQCAccountInfo{}, false
	}
	return info, true
}

// SetPQCAccount stores the PQC account info.
func (k Keeper) SetPQCAccount(ctx sdk.Context, info types.PQCAccountInfo) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(info)
	if err != nil {
		return err
	}
	store.Set(accountKey(info.Address), bz)
	return nil
}

// HasPQCAccount checks whether an account has a registered PQC key.
func (k Keeper) HasPQCAccount(ctx sdk.Context, address string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(accountKey(address))
}

// ---- Stats ----

// GetStats returns the module statistics.
func (k Keeper) GetStats(ctx sdk.Context) types.PQCStats {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.StatsKey)
	if bz == nil {
		return types.PQCStats{}
	}
	var stats types.PQCStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.PQCStats{}
	}
	return stats
}

// SetStats stores the module statistics.
func (k Keeper) SetStats(ctx sdk.Context, stats types.PQCStats) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(stats)
	store.Set(types.StatsKey, bz)
}

// IncrementPQCVerifications increments the PQC verification counter.
func (k Keeper) IncrementPQCVerifications(ctx sdk.Context) {
	stats := k.GetStats(ctx)
	stats.TotalPQCVerifications++
	k.SetStats(ctx, stats)
}

// IncrementClassicalFallbacks increments the classical fallback counter.
func (k Keeper) IncrementClassicalFallbacks(ctx sdk.Context) {
	stats := k.GetStats(ctx)
	stats.TotalClassicalFallbacks++
	k.SetStats(ctx, stats)
}

// ---- Algorithm Registry (v0.6.0) ----

// RegisterAlgorithm adds a new algorithm to the registry.
func (k Keeper) RegisterAlgorithm(ctx sdk.Context, algo types.AlgorithmInfo) error {
	if err := algo.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	key := types.AlgorithmKey(algo.ID)
	if store.Has(key) {
		return types.ErrAlgorithmAlreadyExists.Wrapf("algorithm %d already registered", algo.ID)
	}
	bz, err := json.Marshal(algo)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pqc_algorithm_registered",
		sdk.NewAttribute("algorithm_id", algo.ID.String()),
		sdk.NewAttribute("name", algo.Name),
		sdk.NewAttribute("category", algo.Category),
	))
	return nil
}

// GetAlgorithm returns the algorithm info by ID.
func (k Keeper) GetAlgorithm(ctx sdk.Context, id types.AlgorithmID) (types.AlgorithmInfo, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AlgorithmKey(id))
	if bz == nil {
		return types.AlgorithmInfo{}, types.ErrInvalidAlgorithm.Wrapf("algorithm %d not found", id)
	}
	var algo types.AlgorithmInfo
	if err := json.Unmarshal(bz, &algo); err != nil {
		return types.AlgorithmInfo{}, err
	}
	return algo, nil
}

// ListAlgorithms returns all registered algorithms.
func (k Keeper) ListAlgorithms(ctx sdk.Context) []types.AlgorithmInfo {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AlgorithmPrefix)
	defer iter.Close()

	var algos []types.AlgorithmInfo
	for ; iter.Valid(); iter.Next() {
		var algo types.AlgorithmInfo
		if err := json.Unmarshal(iter.Value(), &algo); err != nil {
			continue
		}
		algos = append(algos, algo)
	}
	return algos
}

// UpdateAlgorithmStatus changes the lifecycle status of an algorithm.
func (k Keeper) UpdateAlgorithmStatus(ctx sdk.Context, id types.AlgorithmID, status types.AlgorithmStatus) error {
	algo, err := k.GetAlgorithm(ctx, id)
	if err != nil {
		return err
	}

	oldStatus := algo.Status
	algo.Status = status

	if status == types.StatusDeprecated {
		algo.DeprecatedAt = ctx.BlockHeight()
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(algo)
	if err != nil {
		return err
	}
	store.Set(types.AlgorithmKey(id), bz)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pqc_algorithm_status_updated",
		sdk.NewAttribute("algorithm_id", id.String()),
		sdk.NewAttribute("old_status", oldStatus.String()),
		sdk.NewAttribute("new_status", status.String()),
	))
	return nil
}

// GetActiveSignatureAlgorithms returns all active signature algorithms.
func (k Keeper) GetActiveSignatureAlgorithms(ctx sdk.Context) []types.AlgorithmInfo {
	var result []types.AlgorithmInfo
	for _, algo := range k.ListAlgorithms(ctx) {
		if algo.Category == types.CategorySignature && algo.Status == types.StatusActive {
			result = append(result, algo)
		}
	}
	return result
}

// GetActiveKEMAlgorithms returns all active KEM algorithms.
func (k Keeper) GetActiveKEMAlgorithms(ctx sdk.Context) []types.AlgorithmInfo {
	var result []types.AlgorithmInfo
	for _, algo := range k.ListAlgorithms(ctx) {
		if algo.Category == types.CategoryKEM && algo.Status == types.StatusActive {
			result = append(result, algo)
		}
	}
	return result
}

// ---- Migration (v0.6.0) ----

// GetMigration returns the migration info for a source algorithm.
func (k Keeper) GetMigration(ctx sdk.Context, fromID types.AlgorithmID) (types.MigrationInfo, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.MigrationKey(fromID))
	if bz == nil {
		return types.MigrationInfo{}, false
	}
	var mig types.MigrationInfo
	if err := json.Unmarshal(bz, &mig); err != nil {
		return types.MigrationInfo{}, false
	}
	return mig, true
}

// SetMigration stores migration info.
func (k Keeper) SetMigration(ctx sdk.Context, migration types.MigrationInfo) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(migration)
	if err != nil {
		return err
	}
	store.Set(types.MigrationKey(migration.FromAlgorithmID), bz)
	return nil
}

// DeleteMigration removes a completed migration.
func (k Keeper) DeleteMigration(ctx sdk.Context, fromID types.AlgorithmID) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.MigrationKey(fromID))
}

// ---- Genesis ----

// InitGenesis initializes the module state from genesis data.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}
	for _, acc := range gs.Accounts {
		if err := k.SetPQCAccount(ctx, acc); err != nil {
			panic(err)
		}
	}
	k.SetStats(ctx, gs.Stats)

	// Register genesis algorithms
	for _, algo := range gs.Algorithms {
		if err := k.RegisterAlgorithm(ctx, algo); err != nil {
			panic(err)
		}
	}

	// Load migrations
	for _, mig := range gs.Migrations {
		if err := k.SetMigration(ctx, mig); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis exports the module state to genesis data.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		Accounts:   k.getAllAccounts(ctx),
		Stats:      k.GetStats(ctx),
		Algorithms: k.ListAlgorithms(ctx),
		Migrations: k.getAllMigrations(ctx),
	}
}

func (k Keeper) getAllAccounts(ctx sdk.Context) []types.PQCAccountInfo {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AccountPrefix)
	defer iter.Close()

	var accounts []types.PQCAccountInfo
	for ; iter.Valid(); iter.Next() {
		var info types.PQCAccountInfo
		if err := json.Unmarshal(iter.Value(), &info); err != nil {
			continue
		}
		accounts = append(accounts, info)
	}
	return accounts
}

func (k Keeper) getAllMigrations(ctx sdk.Context) []types.MigrationInfo {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.MigrationPrefix)
	defer iter.Close()

	var migrations []types.MigrationInfo
	for ; iter.Valid(); iter.Next() {
		var mig types.MigrationInfo
		if err := json.Unmarshal(iter.Value(), &mig); err != nil {
			continue
		}
		migrations = append(migrations, mig)
	}
	return migrations
}
