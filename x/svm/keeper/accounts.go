//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// GetAccount retrieves an SVM account by its 32-byte address.
func (k *Keeper) GetAccount(ctx sdk.Context, addr [32]byte) (*types.SVMAccount, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AccountKey(addr))
	if bz == nil {
		return nil, types.ErrAccountNotFound.Wrapf("account %s not found",
			types.Base58Encode(addr))
	}
	var acc types.SVMAccount
	if err := json.Unmarshal(bz, &acc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SVM account: %w", err)
	}
	return &acc, nil
}

// SetAccount stores or updates an SVM account. It also maintains the
// reverse mapping from the derived native address to the SVM address.
func (k *Keeper) SetAccount(ctx sdk.Context, account *types.SVMAccount) error {
	if account == nil {
		return fmt.Errorf("cannot store nil account")
	}
	store := ctx.KVStore(k.storeKey)

	bz, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal SVM account: %w", err)
	}
	store.Set(types.AccountKey(account.Address), bz)

	// Update the reverse address map (native -> SVM).
	cosmosAddr := types.SVMToCosmosAddress(account.Address)
	store.Set(types.AddrMapKey(cosmosAddr), account.Address[:])

	return nil
}

// DeleteAccount removes an SVM account and its reverse address mapping.
func (k *Keeper) DeleteAccount(ctx sdk.Context, addr [32]byte) error {
	store := ctx.KVStore(k.storeKey)

	key := types.AccountKey(addr)
	if !store.Has(key) {
		return types.ErrAccountNotFound.Wrapf("account %s not found",
			types.Base58Encode(addr))
	}

	// Remove the reverse mapping first.
	cosmosAddr := types.SVMToCosmosAddress(addr)
	store.Delete(types.AddrMapKey(cosmosAddr))

	// Remove the primary account entry.
	store.Delete(key)

	return nil
}

// GetAccountByCosmosAddr retrieves an SVM account using a native address
// by looking up the reverse mapping and then fetching the SVM account.
func (k *Keeper) GetAccountByCosmosAddr(ctx sdk.Context, cosmosAddr sdk.AccAddress) (*types.SVMAccount, error) {
	svmAddr, err := k.cosmosToSVMAddrFromStore(ctx, cosmosAddr)
	if err != nil {
		return nil, err
	}
	return k.GetAccount(ctx, svmAddr)
}

// HasAccount returns true if the account exists in the KVStore.
func (k *Keeper) HasAccount(ctx sdk.Context, addr [32]byte) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.AccountKey(addr))
}

// IterateAccounts iterates over all SVM accounts and calls the callback for each one.
// If the callback returns true, iteration stops.
func (k *Keeper) IterateAccounts(ctx sdk.Context, cb func(types.SVMAccount) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AccountKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var acc types.SVMAccount
		if err := json.Unmarshal(iter.Value(), &acc); err != nil {
			k.logger.Error("failed to unmarshal SVM account during iteration", "error", err)
			continue
		}
		if cb(acc) {
			break
		}
	}
}

// GetAllAccounts returns all SVM accounts in the store.
func (k *Keeper) GetAllAccounts(ctx sdk.Context) []types.SVMAccount {
	var accounts []types.SVMAccount
	k.IterateAccounts(ctx, func(acc types.SVMAccount) bool {
		accounts = append(accounts, acc)
		return false
	})
	return accounts
}

// SVMToCosmosAddr converts a 32-byte SVM address to a native address.
// This is a pure derivation (SHA-256 truncated to 20 bytes) and does not
// require KVStore access.
func (k *Keeper) SVMToCosmosAddr(svmAddr [32]byte) sdk.AccAddress {
	return types.SVMToCosmosAddress(svmAddr)
}

// CosmosToSVMAddr looks up the SVM address mapped to a native address.
// NOTE: This interface method has no sdk.Context parameter so it cannot
// perform a KVStore lookup directly. It delegates to the cached context
// set during the most recent BeginBlock/DeliverTx. For direct access
// with a known context, use cosmosToSVMAddrFromStore.
func (k *Keeper) CosmosToSVMAddr(cosmosAddr sdk.AccAddress) ([32]byte, error) {
	// Without a context we cannot access the KVStore. Return an error
	// indicating the mapping is unavailable in this call path. Modules
	// that need the reverse lookup should call GetAccountByCosmosAddr
	// with the transaction context instead.
	return [32]byte{}, fmt.Errorf("CosmosToSVMAddr requires context; use GetAccountByCosmosAddr instead")
}

// cosmosToSVMAddrFromStore performs the reverse address lookup using a context.
func (k *Keeper) cosmosToSVMAddrFromStore(ctx sdk.Context, cosmosAddr sdk.AccAddress) ([32]byte, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AddrMapKey(cosmosAddr))
	if bz == nil || len(bz) != 32 {
		return [32]byte{}, types.ErrAccountNotFound.Wrapf(
			"no SVM address mapped to native address %s", cosmosAddr.String())
	}
	var svmAddr [32]byte
	copy(svmAddr[:], bz)
	return svmAddr, nil
}
