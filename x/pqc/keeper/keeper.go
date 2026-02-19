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
}

// ExportGenesis exports the module state to genesis data.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:   k.GetParams(ctx),
		Accounts: k.getAllAccounts(ctx),
		Stats:    k.GetStats(ctx),
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
