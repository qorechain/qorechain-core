//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
)

// Keeper manages the SVM module state including accounts, programs, and execution.
type Keeper struct {
	cdc           codec.Codec
	storeKey      storetypes.StoreKey
	logger        log.Logger
	executor      types.SVMExecutor
	pqcKeeper     pqcmod.PQCKeeper
	aiKeeper      aimod.AIKeeper
	crossvmKeeper crossvmmod.CrossVMKeeper
}

// NewKeeper creates a new SVM keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	pqcKeeper pqcmod.PQCKeeper,
	aiKeeper aimod.AIKeeper,
	crossvmKeeper crossvmmod.CrossVMKeeper,
	logger log.Logger,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		logger:        logger.With("module", types.ModuleName),
		pqcKeeper:     pqcKeeper,
		aiKeeper:      aiKeeper,
		crossvmKeeper: crossvmKeeper,
	}
}

// SetExecutor sets the BPF execution engine (called during app init).
func (k *Keeper) SetExecutor(exec types.SVMExecutor) {
	k.executor = exec
}

// Logger returns the keeper's logger.
func (k *Keeper) Logger() log.Logger {
	return k.logger
}

// GetParams reads module parameters from the KVStore.
func (k *Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		k.logger.Error("failed to unmarshal svm params", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams writes module parameters to the KVStore.
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid svm params: %w", err)
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal svm params: %w", err)
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// GetCurrentSlot returns the current SVM slot number from the KVStore.
func (k *Keeper) GetCurrentSlot(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SlotKey)
	if bz == nil || len(bz) < 8 {
		return 0
	}
	return uint64(bz[0]) | uint64(bz[1])<<8 | uint64(bz[2])<<16 | uint64(bz[3])<<24 |
		uint64(bz[4])<<32 | uint64(bz[5])<<40 | uint64(bz[6])<<48 | uint64(bz[7])<<56
}

// SetCurrentSlot stores the current SVM slot number.
func (k *Keeper) SetCurrentSlot(ctx sdk.Context, slot uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	bz[0] = byte(slot)
	bz[1] = byte(slot >> 8)
	bz[2] = byte(slot >> 16)
	bz[3] = byte(slot >> 24)
	bz[4] = byte(slot >> 32)
	bz[5] = byte(slot >> 40)
	bz[6] = byte(slot >> 48)
	bz[7] = byte(slot >> 56)
	store.Set(types.SlotKey, bz)
}
