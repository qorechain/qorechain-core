//go:build proprietary

package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/inflation/types"
)

// BankKeeper defines the expected bank keeper interface for the inflation module.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// Keeper manages the inflation module state.
type Keeper struct {
	cdc        codec.Codec
	storeKey   storetypes.StoreKey
	bankKeeper BankKeeper
	logger     log.Logger
}

// NewKeeper creates a new inflation keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bankKeeper BankKeeper,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		logger:     logger.With("module", types.ModuleName),
	}
}

// Logger returns the module logger.
func (k Keeper) Logger() log.Logger { return k.logger }

// --- Params ---

// GetParams returns the inflation module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		k.logger.Warn("failed to unmarshal inflation params, using defaults", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the inflation module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// --- Epoch Info ---

// GetEpochInfo returns the current epoch state.
func (k Keeper) GetEpochInfo(ctx sdk.Context) types.EpochInfo {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CurrentEpochKey)
	if bz == nil {
		return types.DefaultEpochInfo()
	}
	var info types.EpochInfo
	if err := json.Unmarshal(bz, &info); err != nil {
		k.logger.Warn("failed to unmarshal epoch info, using defaults", "error", err)
		return types.DefaultEpochInfo()
	}
	return info
}

// setEpochInfo stores the epoch info.
func (k Keeper) setEpochInfo(ctx sdk.Context, info types.EpochInfo) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(info)
	if err != nil {
		return err
	}
	store.Set(types.CurrentEpochKey, bz)
	return nil
}

// --- Inflation Rate ---

// GetCurrentInflationRate finds the schedule tier for the current year.
// Walks the schedule and returns the last tier where tier.Year <= currentYear.
// If the current year exceeds all tiers, uses the last tier's rate (perpetual).
func (k Keeper) GetCurrentInflationRate(ctx sdk.Context) math.LegacyDec {
	params := k.GetParams(ctx)
	epochInfo := k.GetEpochInfo(ctx)

	if len(params.Schedule) == 0 {
		return math.LegacyZeroDec()
	}

	// Default to the first tier's rate
	rate := params.Schedule[0].InflationRate
	for _, tier := range params.Schedule {
		if tier.Year <= epochInfo.CurrentYear {
			rate = tier.InflationRate
		} else {
			break
		}
	}
	return rate
}

// --- Core Emission ---

// MintEpochEmission mints new tokens when a new epoch begins and sends them
// to fee_collector for staking distribution.
func (k Keeper) MintEpochEmission(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return nil
	}

	epochInfo := k.GetEpochInfo(ctx)

	// Check if current block triggers a new epoch
	if ctx.BlockHeight() < epochInfo.BlockStart+params.EpochLength {
		return nil
	}

	// Get current inflation rate
	inflationRate := k.GetCurrentInflationRate(ctx)
	if inflationRate.IsZero() {
		// Update epoch tracking even if rate is zero
		epochInfo.CurrentEpoch++
		epochInfo.BlockStart = ctx.BlockHeight()
		if epochInfo.CurrentEpoch > 0 && epochInfo.CurrentEpoch%uint64(365) == 0 {
			epochInfo.CurrentYear++
		}
		if err := k.setEpochInfo(ctx, epochInfo); err != nil {
			return fmt.Errorf("failed to set epoch info: %w", err)
		}
		return nil
	}

	// Calculate emission: totalSupply * inflationRate / epochsPerYear
	const epochsPerYear = int64(365)
	totalSupply := k.bankKeeper.GetSupply(ctx, "uqor").Amount
	if totalSupply.IsZero() {
		return nil
	}

	epochEmission := inflationRate.MulInt(totalSupply).QuoInt64(epochsPerYear).TruncateInt()
	if !epochEmission.IsPositive() {
		return nil
	}

	// Mint coins to inflation module account
	mintCoins := sdk.NewCoins(sdk.NewCoin("uqor", epochEmission))
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, mintCoins); err != nil {
		return fmt.Errorf("failed to mint epoch emission: %w", err)
	}

	// Send minted coins to fee_collector for staking distribution
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "fee_collector", mintCoins); err != nil {
		return fmt.Errorf("failed to send emission to fee_collector: %w", err)
	}

	// Update epoch info
	epochInfo.CurrentEpoch++
	epochInfo.BlockStart = ctx.BlockHeight()
	epochInfo.TotalMinted = epochInfo.TotalMinted.Add(epochEmission)

	// Check if year changed
	if epochInfo.CurrentEpoch > 0 && epochInfo.CurrentEpoch%uint64(epochsPerYear) == 0 {
		epochInfo.CurrentYear++
	}

	if err := k.setEpochInfo(ctx, epochInfo); err != nil {
		return fmt.Errorf("failed to set epoch info: %w", err)
	}

	k.logger.Info("minted epoch emission",
		"epoch", epochInfo.CurrentEpoch,
		"year", epochInfo.CurrentYear,
		"emission", epochEmission.String(),
		"total_minted", epochInfo.TotalMinted.String(),
		"inflation_rate", inflationRate.String(),
		"height", ctx.BlockHeight(),
	)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"epoch_emission",
			sdk.NewAttribute("epoch", fmt.Sprintf("%d", epochInfo.CurrentEpoch)),
			sdk.NewAttribute("year", fmt.Sprintf("%d", epochInfo.CurrentYear)),
			sdk.NewAttribute("emission", epochEmission.String()),
			sdk.NewAttribute("inflation_rate", inflationRate.String()),
		),
	)

	return nil
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set inflation params: %v", err))
	}
	if err := k.setEpochInfo(ctx, gs.EpochInfo); err != nil {
		panic(fmt.Sprintf("failed to set inflation epoch info: %v", err))
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:    k.GetParams(ctx),
		EpochInfo: k.GetEpochInfo(ctx),
	}
}
