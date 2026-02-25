//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// Keeper manages the gasabstraction module state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	logger   log.Logger
}

// NewKeeper creates a new gas abstraction keeper.
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

// GetConfig returns the gas abstraction module configuration.
func (k Keeper) GetConfig(ctx sdk.Context) types.GasAbstractionConfig {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ConfigKey)
	if bz == nil {
		return types.DefaultGasAbstractionConfig()
	}
	var config types.GasAbstractionConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		k.logger.Warn("failed to unmarshal gasabstraction config, using defaults", "error", err)
		return types.DefaultGasAbstractionConfig()
	}
	return config
}

// SetConfig stores the gas abstraction module configuration.
func (k Keeper) SetConfig(ctx sdk.Context, config types.GasAbstractionConfig) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(config)
	if err != nil {
		return err
	}
	store.Set(types.ConfigKey, bz)
	return nil
}

// IsEnabled returns whether gas abstraction is enabled.
func (k Keeper) IsEnabled(ctx sdk.Context) bool {
	return k.GetConfig(ctx).Enabled
}

// GetAcceptedTokens returns the list of accepted fee tokens.
func (k Keeper) GetAcceptedTokens(ctx sdk.Context) []types.AcceptedFeeToken {
	return k.GetConfig(ctx).AcceptedTokens
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetConfig(ctx, gs.Config); err != nil {
		panic(fmt.Sprintf("failed to set gasabstraction config: %v", err))
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Config: k.GetConfig(ctx),
	}
}
