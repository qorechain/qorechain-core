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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/qorechain/qorechain-core/x/burn/types"
)

// BankKeeper defines the expected bank keeper interface for the burn module.
type BankKeeper interface {
	BurnCoins(ctx context.Context, moduleName string, coins sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// Keeper manages the burn module state.
type Keeper struct {
	cdc        codec.Codec
	storeKey   storetypes.StoreKey
	bankKeeper BankKeeper
	logger     log.Logger
}

// NewKeeper creates a new burn keeper.
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

// GetParams returns the burn module parameters.
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

// SetParams stores the burn module parameters.
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

// --- Burn Stats ---

// GetBurnStats returns the aggregate burn statistics.
func (k Keeper) GetBurnStats(ctx sdk.Context) types.BurnStats {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalBurnedKey)
	if bz == nil {
		return types.DefaultBurnStats()
	}
	var stats types.BurnStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return types.DefaultBurnStats()
	}
	return stats
}

func (k Keeper) setBurnStats(ctx sdk.Context, stats types.BurnStats) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	store.Set(types.TotalBurnedKey, bz)
	return nil
}

// GetTotalBurned returns the total amount of uqor burned.
func (k Keeper) GetTotalBurned(ctx sdk.Context) math.Int {
	return k.GetBurnStats(ctx).TotalBurned
}

// --- Burn Records ---

func (k Keeper) addBurnRecord(ctx sdk.Context, record types.BurnRecord) error {
	store := ctx.KVStore(k.storeKey)
	key := append(types.BurnRecordPrefix, []byte(fmt.Sprintf("%020d/%s", record.Height, record.Source))...)
	bz, err := json.Marshal(record)
	if err != nil {
		return err
	}
	store.Set(key, bz)
	return nil
}

// GetBurnRecords returns the most recent burn records up to limit.
func (k Keeper) GetBurnRecords(ctx sdk.Context, limit int) []types.BurnRecord {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStoreReversePrefixIterator(store, types.BurnRecordPrefix)
	defer iter.Close()

	var records []types.BurnRecord
	for ; iter.Valid() && len(records) < limit; iter.Next() {
		var record types.BurnRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	return records
}

// --- Core Burn ---

// BurnFromSource burns coins and records the event.
func (k Keeper) BurnFromSource(ctx sdk.Context, source types.BurnSource, amount math.Int, txHash string) error {
	if !types.IsValidBurnSource(source) {
		return types.ErrInvalidBurnSource
	}
	if !amount.IsPositive() {
		return types.ErrInvalidBurnAmount
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.ErrBurnDisabled
	}

	// Burn from module account
	coins := sdk.NewCoins(sdk.NewCoin("uqor", amount))
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return fmt.Errorf("failed to burn coins: %w", err)
	}

	// Update stats
	stats := k.GetBurnStats(ctx)
	stats.TotalBurned = stats.TotalBurned.Add(amount)
	if stats.BurnsBySource == nil {
		stats.BurnsBySource = make(map[types.BurnSource]math.Int)
	}
	prev, ok := stats.BurnsBySource[source]
	if !ok {
		prev = math.ZeroInt()
	}
	stats.BurnsBySource[source] = prev.Add(amount)
	stats.LastBurnHeight = ctx.BlockHeight()
	if err := k.setBurnStats(ctx, stats); err != nil {
		return err
	}

	// Record
	record := types.BurnRecord{
		Source: source,
		Amount: amount,
		Height: ctx.BlockHeight(),
		TxHash: txHash,
	}
	if err := k.addBurnRecord(ctx, record); err != nil {
		return err
	}

	k.logger.Info("burned coins",
		"source", source,
		"amount", amount.String(),
		"height", ctx.BlockHeight(),
	)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"burn",
			sdk.NewAttribute("source", string(source)),
			sdk.NewAttribute("amount", amount.String()),
		),
	)

	return nil
}

// --- Fee Distribution (EndBlocker) ---

// DistributeFees splits the fee collector balance according to params.
// Called in EndBlocker each block.
func (k Keeper) DistributeFees(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return nil
	}

	// Get fee collector balance
	feeCollectorAddr := authtypes.NewModuleAddress("fee_collector")
	balance := k.bankKeeper.GetBalance(ctx, feeCollectorAddr, "uqor")
	if balance.IsZero() {
		return nil
	}

	totalFees := balance.Amount

	// Calculate shares
	burnAmount := params.GasBurnRate.MulInt(totalFees).TruncateInt()
	treasuryAmount := params.TreasuryShare.MulInt(totalFees).TruncateInt()
	stakerAmount := params.StakerShare.MulInt(totalFees).TruncateInt()
	// Validator share is the remainder (avoids rounding issues)
	validatorAmount := totalFees.Sub(burnAmount).Sub(treasuryAmount).Sub(stakerAmount)

	// Send burn portion to burn module, then burn it
	if burnAmount.IsPositive() {
		burnCoins := sdk.NewCoins(sdk.NewCoin("uqor", burnAmount))
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, "fee_collector", types.ModuleName, burnCoins); err != nil {
			k.logger.Error("failed to send fees to burn module", "error", err)
		} else {
			if err := k.BurnFromSource(ctx, types.BurnSourceGasFee, burnAmount, ""); err != nil {
				k.logger.Error("failed to burn gas fees", "error", err)
			}
		}
	}

	// Send treasury portion to protocol pool
	if treasuryAmount.IsPositive() {
		treasuryCoins := sdk.NewCoins(sdk.NewCoin("uqor", treasuryAmount))
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, "fee_collector", "protocolpool", treasuryCoins); err != nil {
			k.logger.Error("failed to send fees to treasury", "error", err)
		}
	}

	// Staker share stays in fee_collector for distribution module to handle
	// Validator share also stays in fee_collector (default distribution flow)
	_ = validatorAmount
	_ = stakerAmount

	return nil
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set burn params: %v", err))
	}
	if err := k.setBurnStats(ctx, gs.Stats); err != nil {
		panic(fmt.Sprintf("failed to set burn stats: %v", err))
	}
	for _, r := range gs.Records {
		if err := k.addBurnRecord(ctx, r); err != nil {
			panic(fmt.Sprintf("failed to add burn record: %v", err))
		}
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:  k.GetParams(ctx),
		Stats:   k.GetBurnStats(ctx),
		Records: k.GetBurnRecords(ctx, 10000),
	}
}
