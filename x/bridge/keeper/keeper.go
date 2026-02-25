//go:build proprietary

package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	pqckeeper "github.com/qorechain/qorechain-core/x/pqc/keeper"
	"github.com/qorechain/qorechain-core/x/bridge/types"
	burntypes "github.com/qorechain/qorechain-core/x/burn/types"
)

// BridgeBurnKeeper defines what the bridge needs from the burn module.
type BridgeBurnKeeper interface {
	BurnFromSource(ctx sdk.Context, source burntypes.BurnSource, amount math.Int, txHash string) error
}

// Keeper manages the x/bridge module state.
type Keeper struct {
	cdc        codec.Codec
	storeKey   storetypes.StoreKey
	pqcKeeper  pqckeeper.Keeper
	burnKeeper BridgeBurnKeeper // may be nil if burn module not wired
	logger     log.Logger
}

// NewKeeper creates a new bridge keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	pqcKeeper pqckeeper.Keeper,
	burnKeeper BridgeBurnKeeper,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		pqcKeeper:  pqcKeeper,
		burnKeeper: burnKeeper,
		logger:     logger.With("module", types.ModuleName),
	}
}

// BurnWithdrawalFee burns the bridge withdrawal fee via the burn module.
// Non-fatal: logs a warning on failure so withdrawals are never blocked.
func (k Keeper) BurnWithdrawalFee(ctx sdk.Context, amount math.Int, txHash string) {
	if k.burnKeeper == nil || amount.IsZero() {
		return
	}
	if err := k.burnKeeper.BurnFromSource(ctx, burntypes.BurnSourceBridgeFee, amount, txHash); err != nil {
		k.logger.Warn("bridge burn fee failed (non-fatal)", "amount", amount.String(), "error", err)
	}
}

// Logger returns the module logger.
func (k Keeper) Logger() log.Logger {
	return k.logger
}

// ---- Config ----

// GetConfig returns the bridge configuration.
func (k Keeper) GetConfig(ctx sdk.Context) types.BridgeConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConfigKey)
	if bz == nil {
		return types.DefaultBridgeConfig()
	}
	var config types.BridgeConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return types.DefaultBridgeConfig()
	}
	return config
}

// SetConfig stores the bridge configuration.
func (k Keeper) SetConfig(ctx sdk.Context, config types.BridgeConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(config)
	if err != nil {
		return err
	}
	store.Set(types.ConfigKey, bz)
	return nil
}

// ---- Chain Configs ----

func chainKey(chainID string) []byte {
	return append(types.ChainConfigPrefix, []byte(chainID)...)
}

// GetChainConfig returns the configuration for a specific chain.
func (k Keeper) GetChainConfig(ctx sdk.Context, chainID string) (types.ChainConfig, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(chainKey(chainID))
	if bz == nil {
		return types.ChainConfig{}, false
	}
	var cc types.ChainConfig
	if err := json.Unmarshal(bz, &cc); err != nil {
		return types.ChainConfig{}, false
	}
	return cc, true
}

// SetChainConfig stores the configuration for a specific chain.
func (k Keeper) SetChainConfig(ctx sdk.Context, cc types.ChainConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(cc)
	if err != nil {
		return err
	}
	store.Set(chainKey(cc.ChainID), bz)
	return nil
}

// GetAllChainConfigs returns all chain configurations.
func (k Keeper) GetAllChainConfigs(ctx sdk.Context) []types.ChainConfig {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ChainConfigPrefix)
	defer iter.Close()

	var configs []types.ChainConfig
	for ; iter.Valid(); iter.Next() {
		var cc types.ChainConfig
		if err := json.Unmarshal(iter.Value(), &cc); err != nil {
			continue
		}
		configs = append(configs, cc)
	}
	return configs
}

// ---- Bridge Validators ----

func validatorKey(address string) []byte {
	return append(types.ValidatorPrefix, []byte(address)...)
}

// GetBridgeValidator returns a bridge validator by address.
func (k Keeper) GetBridgeValidator(ctx sdk.Context, address string) (types.BridgeValidator, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(validatorKey(address))
	if bz == nil {
		return types.BridgeValidator{}, false
	}
	var v types.BridgeValidator
	if err := json.Unmarshal(bz, &v); err != nil {
		return types.BridgeValidator{}, false
	}
	return v, true
}

// SetBridgeValidator stores a bridge validator.
func (k Keeper) SetBridgeValidator(ctx sdk.Context, v types.BridgeValidator) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(v)
	if err != nil {
		return err
	}
	store.Set(validatorKey(v.Address), bz)
	return nil
}

// GetAllBridgeValidators returns all registered bridge validators.
func (k Keeper) GetAllBridgeValidators(ctx sdk.Context) []types.BridgeValidator {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ValidatorPrefix)
	defer iter.Close()

	var validators []types.BridgeValidator
	for ; iter.Valid(); iter.Next() {
		var v types.BridgeValidator
		if err := json.Unmarshal(iter.Value(), &v); err != nil {
			continue
		}
		validators = append(validators, v)
	}
	return validators
}

// GetActiveValidatorsForChain returns active validators supporting a specific chain.
func (k Keeper) GetActiveValidatorsForChain(ctx sdk.Context, chainID string) []types.BridgeValidator {
	all := k.GetAllBridgeValidators(ctx)
	var result []types.BridgeValidator
	for _, v := range all {
		if !v.Active {
			continue
		}
		for _, c := range v.SupportedChains {
			if c == chainID {
				result = append(result, v)
				break
			}
		}
	}
	return result
}

// ---- Operations ----

func operationKey(id string) []byte {
	return append(types.OperationPrefix, []byte(id)...)
}

// GetOperation returns a bridge operation by ID.
func (k Keeper) GetOperation(ctx sdk.Context, id string) (types.BridgeOperation, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(operationKey(id))
	if bz == nil {
		return types.BridgeOperation{}, false
	}
	var op types.BridgeOperation
	if err := json.Unmarshal(bz, &op); err != nil {
		return types.BridgeOperation{}, false
	}
	return op, true
}

// SetOperation stores a bridge operation.
func (k Keeper) SetOperation(ctx sdk.Context, op types.BridgeOperation) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(op)
	if err != nil {
		return err
	}
	store.Set(operationKey(op.ID), bz)
	return nil
}

// GetAllOperations returns all bridge operations.
func (k Keeper) GetAllOperations(ctx sdk.Context) []types.BridgeOperation {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.OperationPrefix)
	defer iter.Close()

	var ops []types.BridgeOperation
	for ; iter.Valid(); iter.Next() {
		var op types.BridgeOperation
		if err := json.Unmarshal(iter.Value(), &op); err != nil {
			continue
		}
		ops = append(ops, op)
	}
	return ops
}

// NextOperationID generates the next unique operation ID.
func (k Keeper) NextOperationID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.OperationCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.OperationCounterKey, newBz)
	return fmt.Sprintf("OP-%d-%d", ctx.BlockHeight(), counter)
}

// ---- Locked Amounts ----

func lockedKey(chain, asset string) []byte {
	return append(types.LockedAmountPrefix, []byte(chain+"/"+asset)...)
}

// GetLockedAmount returns the locked amount for a chain/asset pair.
func (k Keeper) GetLockedAmount(ctx sdk.Context, chain, asset string) types.LockedAmount {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(lockedKey(chain, asset))
	if bz == nil {
		return types.LockedAmount{
			Chain:       chain,
			Asset:       asset,
			TotalLocked: "0",
			TotalMinted: "0",
		}
	}
	var la types.LockedAmount
	if err := json.Unmarshal(bz, &la); err != nil {
		return types.LockedAmount{Chain: chain, Asset: asset, TotalLocked: "0", TotalMinted: "0"}
	}
	return la
}

// SetLockedAmount stores the locked amount for a chain/asset pair.
func (k Keeper) SetLockedAmount(ctx sdk.Context, la types.LockedAmount) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(la)
	if err != nil {
		return err
	}
	store.Set(lockedKey(la.Chain, la.Asset), bz)
	return nil
}

// GetAllLockedAmounts returns all locked amounts.
func (k Keeper) GetAllLockedAmounts(ctx sdk.Context) []types.LockedAmount {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.LockedAmountPrefix)
	defer iter.Close()

	var amounts []types.LockedAmount
	for ; iter.Valid(); iter.Next() {
		var la types.LockedAmount
		if err := json.Unmarshal(iter.Value(), &la); err != nil {
			continue
		}
		amounts = append(amounts, la)
	}
	return amounts
}

// ---- Circuit Breakers ----

func breakerKey(chain string) []byte {
	return append(types.CircuitBreakerPrefix, []byte(chain)...)
}

// GetCircuitBreaker returns the circuit breaker state for a chain.
func (k Keeper) GetCircuitBreaker(ctx sdk.Context, chain string) types.CircuitBreakerState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(breakerKey(chain))
	if bz == nil {
		return types.CircuitBreakerState{
			Chain:             chain,
			MaxSingleTransfer: "1000000000000",
			DailyLimit:        "10000000000000",
			CurrentDaily:      "0",
			LastResetHeight:   0,
		}
	}
	var cb types.CircuitBreakerState
	if err := json.Unmarshal(bz, &cb); err != nil {
		return types.CircuitBreakerState{Chain: chain, MaxSingleTransfer: "1000000000000", DailyLimit: "10000000000000", CurrentDaily: "0"}
	}
	return cb
}

// SetCircuitBreaker stores the circuit breaker state for a chain.
func (k Keeper) SetCircuitBreaker(ctx sdk.Context, cb types.CircuitBreakerState) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(cb)
	if err != nil {
		return err
	}
	store.Set(breakerKey(cb.Chain), bz)
	return nil
}

// GetAllCircuitBreakers returns all circuit breaker states.
func (k Keeper) GetAllCircuitBreakers(ctx sdk.Context) []types.CircuitBreakerState {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.CircuitBreakerPrefix)
	defer iter.Close()

	var breakers []types.CircuitBreakerState
	for ; iter.Valid(); iter.Next() {
		var cb types.CircuitBreakerState
		if err := json.Unmarshal(iter.Value(), &cb); err != nil {
			continue
		}
		breakers = append(breakers, cb)
	}
	return breakers
}

// ---- Genesis ----

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetConfig(ctx, gs.Config); err != nil {
		panic(err)
	}
	for _, cc := range gs.ChainConfigs {
		if err := k.SetChainConfig(ctx, cc); err != nil {
			panic(err)
		}
	}
	for _, v := range gs.Validators {
		if err := k.SetBridgeValidator(ctx, v); err != nil {
			panic(err)
		}
	}
	for _, op := range gs.Operations {
		if err := k.SetOperation(ctx, op); err != nil {
			panic(err)
		}
	}
	for _, la := range gs.Locked {
		if err := k.SetLockedAmount(ctx, la); err != nil {
			panic(err)
		}
	}
	for _, cb := range gs.Breakers {
		if err := k.SetCircuitBreaker(ctx, cb); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Config:       k.GetConfig(ctx),
		ChainConfigs: k.GetAllChainConfigs(ctx),
		Validators:   k.GetAllBridgeValidators(ctx),
		Operations:   k.GetAllOperations(ctx),
		Locked:       k.GetAllLockedAmounts(ctx),
		Breakers:     k.GetAllCircuitBreakers(ctx),
	}
}
