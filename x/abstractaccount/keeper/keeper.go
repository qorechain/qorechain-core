//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// Keeper manages the abstractaccount module state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	logger   log.Logger
}

// NewKeeper creates a new abstract account keeper.
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

// GetConfig returns the abstract account configuration.
func (k Keeper) GetConfig(ctx sdk.Context) types.AbstractAccountConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConfigKey)
	if bz == nil {
		return types.DefaultAbstractAccountConfig()
	}
	var config types.AbstractAccountConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		k.logger.Warn("failed to unmarshal abstractaccount config, using defaults", "error", err)
		return types.DefaultAbstractAccountConfig()
	}
	return config
}

// SetConfig stores the abstract account configuration.
func (k Keeper) SetConfig(ctx sdk.Context, config types.AbstractAccountConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(config)
	if err != nil {
		return err
	}
	store.Set(types.ConfigKey, bz)
	return nil
}

// IsEnabled returns whether abstract accounts are enabled.
func (k Keeper) IsEnabled(ctx sdk.Context) bool {
	return k.GetConfig(ctx).Enabled
}

// --- Accounts ---

func accountKey(address string) []byte {
	return append(types.AccountPrefix, []byte(address)...)
}

// GetAccount returns an abstract account by address.
func (k Keeper) GetAccount(ctx sdk.Context, address string) (types.AbstractAccount, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(accountKey(address))
	if bz == nil {
		return types.AbstractAccount{}, false
	}
	var acc types.AbstractAccount
	if err := json.Unmarshal(bz, &acc); err != nil {
		return types.AbstractAccount{}, false
	}
	return acc, true
}

// SetAccount stores an abstract account.
func (k Keeper) SetAccount(ctx sdk.Context, acc types.AbstractAccount) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	store.Set(accountKey(acc.Address), bz)
	return nil
}

// GetAllAccounts returns all abstract accounts.
func (k Keeper) GetAllAccounts(ctx sdk.Context) []types.AbstractAccount {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AccountPrefix)
	defer iter.Close()

	var accounts []types.AbstractAccount
	for ; iter.Valid(); iter.Next() {
		var acc types.AbstractAccount
		if err := json.Unmarshal(iter.Value(), &acc); err != nil {
			continue
		}
		accounts = append(accounts, acc)
	}
	return accounts
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetConfig(ctx, gs.Config); err != nil {
		panic(fmt.Sprintf("failed to set abstractaccount config: %v", err))
	}
	for _, acc := range gs.Accounts {
		if err := k.SetAccount(ctx, acc); err != nil {
			panic(fmt.Sprintf("failed to set abstract account: %v", err))
		}
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Config:   k.GetConfig(ctx),
		Accounts: k.GetAllAccounts(ctx),
	}
}
