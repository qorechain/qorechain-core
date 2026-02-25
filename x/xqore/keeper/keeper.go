//go:build proprietary

package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/xqore/types"
)

// BankKeeper defines the expected bank keeper interface for the xqore module.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// Keeper manages the xqore module state.
type Keeper struct {
	cdc        codec.Codec
	storeKey   storetypes.StoreKey
	bankKeeper BankKeeper
	logger     log.Logger
}

// NewKeeper creates a new xqore keeper.
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

// GetParams returns the xqore module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		k.logger.Warn("failed to unmarshal xqore params, using defaults", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the xqore module parameters.
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

// --- Position Management ---

// positionKey returns the store key for a position owner (safe byte construction).
func positionKey(owner string) []byte {
	ownerBytes := []byte(owner)
	key := make([]byte, 0, len(types.PositionPrefix)+len(ownerBytes))
	key = append(key, types.PositionPrefix...)
	key = append(key, ownerBytes...)
	return key
}

// GetPosition returns the xQORE position for an owner.
func (k Keeper) GetPosition(ctx sdk.Context, owner sdk.AccAddress) (types.XQOREPosition, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(positionKey(owner.String()))
	if bz == nil {
		return types.XQOREPosition{}, false
	}
	var pos types.XQOREPosition
	if err := json.Unmarshal(bz, &pos); err != nil {
		k.logger.Warn("failed to unmarshal xqore position, returning empty", "error", err)
		return types.XQOREPosition{}, false
	}
	return pos, true
}

// setPosition stores a position in the KV store.
func (k Keeper) setPosition(ctx sdk.Context, pos types.XQOREPosition) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(pos)
	if err != nil {
		return err
	}
	store.Set(positionKey(pos.Owner), bz)
	return nil
}

// deletePosition removes a position from the KV store.
func (k Keeper) deletePosition(ctx sdk.Context, owner string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(positionKey(owner))
}

// GetAllPositions iterates all positions in the store.
func (k Keeper) GetAllPositions(ctx sdk.Context) []types.XQOREPosition {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.PositionPrefix)
	defer iter.Close()

	positions := make([]types.XQOREPosition, 0)
	for ; iter.Valid(); iter.Next() {
		var pos types.XQOREPosition
		if err := json.Unmarshal(iter.Value(), &pos); err != nil {
			continue
		}
		positions = append(positions, pos)
	}
	return positions
}

// --- Totals ---

// GetTotalLocked returns the total QORE locked in xQORE positions.
func (k Keeper) GetTotalLocked(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalLockedKey)
	if bz == nil {
		return math.ZeroInt()
	}
	var total math.Int
	if err := json.Unmarshal(bz, &total); err != nil {
		k.logger.Warn("failed to unmarshal total locked, returning zero", "error", err)
		return math.ZeroInt()
	}
	return total
}

func (k Keeper) setTotalLocked(ctx sdk.Context, total math.Int) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(total)
	if err != nil {
		return err
	}
	store.Set(types.TotalLockedKey, bz)
	return nil
}

// GetTotalXQORESupply returns the total xQORE supply.
func (k Keeper) GetTotalXQORESupply(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TotalXQOREKey)
	if bz == nil {
		return math.ZeroInt()
	}
	var total math.Int
	if err := json.Unmarshal(bz, &total); err != nil {
		k.logger.Warn("failed to unmarshal total xqore supply, returning zero", "error", err)
		return math.ZeroInt()
	}
	return total
}

func (k Keeper) setTotalXQORESupply(ctx sdk.Context, total math.Int) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(total)
	if err != nil {
		return err
	}
	store.Set(types.TotalXQOREKey, bz)
	return nil
}

// --- Balance Query ---

// GetXQOREBalance returns the xQORE balance for an address.
// Satisfies the rlconsensusmod.TokenomicsKeeper interface.
func (k Keeper) GetXQOREBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int {
	pos, found := k.GetPosition(ctx, addr)
	if !found {
		return math.ZeroInt()
	}
	return pos.XBalance
}

// --- Governance Multiplier ---

// GetGovernanceMultiplier returns the governance voting power multiplier for an address.
// If the address has an xQORE position, returns the configured multiplier; otherwise 1.0.
func (k Keeper) GetGovernanceMultiplier(ctx sdk.Context, addr sdk.AccAddress) math.LegacyDec {
	_, found := k.GetPosition(ctx, addr)
	if !found {
		return math.LegacyOneDec()
	}
	params := k.GetParams(ctx)
	return params.GovernanceMultiplier
}

// --- Core Operations ---

// Lock locks QORE and mints xQORE 1:1.
func (k Keeper) Lock(ctx sdk.Context, owner sdk.AccAddress, amount math.Int) error {
	if !amount.IsPositive() {
		return types.ErrInvalidLockAmount
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.ErrModuleDisabled
	}
	if amount.LT(params.MinLockAmount) {
		return types.ErrMinLockAmount
	}

	// Send QORE from user to xqore module account
	coins := sdk.NewCoins(sdk.NewCoin("uqor", amount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, owner, types.ModuleName, coins); err != nil {
		return fmt.Errorf("failed to send coins to xqore module: %w", err)
	}

	// Update or create position
	pos, found := k.GetPosition(ctx, owner)
	if found {
		// Note: Re-locking adds to an existing position without resetting LockTime.
		// This means the penalty schedule is calculated from the original lock time,
		// which benefits users who add funds to mature positions. This is by design
		// to incentivize early and sustained participation.
		pos.Locked = pos.Locked.Add(amount)
		pos.XBalance = pos.XBalance.Add(amount)
	} else {
		pos = types.XQOREPosition{
			Owner:      owner.String(),
			Locked:     amount,
			XBalance:   amount,
			LockHeight: ctx.BlockHeight(),
			LockTime:   ctx.BlockTime(),
		}
	}
	if err := k.setPosition(ctx, pos); err != nil {
		return fmt.Errorf("failed to set position: %w", err)
	}

	// Update totals
	totalLocked := k.GetTotalLocked(ctx).Add(amount)
	if err := k.setTotalLocked(ctx, totalLocked); err != nil {
		return fmt.Errorf("failed to set total locked: %w", err)
	}
	totalSupply := k.GetTotalXQORESupply(ctx).Add(amount)
	if err := k.setTotalXQORESupply(ctx, totalSupply); err != nil {
		return fmt.Errorf("failed to set total xqore supply: %w", err)
	}

	k.logger.Info("locked QORE for xQORE",
		"owner", owner.String(),
		"amount", amount.String(),
		"height", ctx.BlockHeight(),
	)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"xqore_lock",
			sdk.NewAttribute("owner", owner.String()),
			sdk.NewAttribute("amount", amount.String()),
		),
	)

	return nil
}

// Unlock redeems xQORE back to QORE with graduated exit penalties.
func (k Keeper) Unlock(ctx sdk.Context, owner sdk.AccAddress, amount math.Int) error {
	if !amount.IsPositive() {
		return types.ErrInvalidLockAmount
	}

	pos, found := k.GetPosition(ctx, owner)
	if !found {
		return types.ErrPositionNotFound
	}
	if amount.GT(pos.XBalance) {
		return types.ErrInsufficientBalance
	}

	params := k.GetParams(ctx)
	if !params.Enabled {
		return types.ErrModuleDisabled
	}

	// Calculate penalty based on time since lock
	elapsed := ctx.BlockTime().Sub(pos.LockTime)
	penaltyRate := k.findPenaltyRate(params.ExitPenaltySchedule, elapsed)

	// penaltyAmount = penaltyRate * amount
	penaltyAmount := penaltyRate.MulInt(amount).TruncateInt()
	// burnAmount = penaltyAmount * PenaltyBurnRate (50% of penalty burned)
	burnAmount := params.PenaltyBurnRate.MulInt(penaltyAmount).TruncateInt()
	// redistAmount = penaltyAmount - burnAmount (rest to remaining holders)
	redistAmount := penaltyAmount.Sub(burnAmount)
	// returnAmount = amount - penaltyAmount
	returnAmount := amount.Sub(penaltyAmount)

	// Burn the penalty burn portion
	if burnAmount.IsPositive() {
		burnCoins := sdk.NewCoins(sdk.NewCoin("uqor", burnAmount))
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
			return fmt.Errorf("failed to burn penalty coins: %w", err)
		}
	}

	// Return remaining amount to user
	if returnAmount.IsPositive() {
		returnCoins := sdk.NewCoins(sdk.NewCoin("uqor", returnAmount))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, owner, returnCoins); err != nil {
			return fmt.Errorf("failed to return coins to user: %w", err)
		}
	}

	// Update position
	pos.XBalance = pos.XBalance.Sub(amount)
	pos.Locked = pos.Locked.Sub(amount)

	// If position is zero, delete it; otherwise update
	if pos.XBalance.IsZero() && pos.Locked.IsZero() {
		k.deletePosition(ctx, pos.Owner)
	} else {
		if err := k.setPosition(ctx, pos); err != nil {
			return fmt.Errorf("failed to update position: %w", err)
		}
	}

	// Update totals
	totalLocked := k.GetTotalLocked(ctx).Sub(amount)
	if err := k.setTotalLocked(ctx, totalLocked); err != nil {
		return fmt.Errorf("failed to set total locked: %w", err)
	}
	// xQORE supply decreases by amount (the full amount, not just return)
	totalSupply := k.GetTotalXQORESupply(ctx).Sub(amount)
	if err := k.setTotalXQORESupply(ctx, totalSupply); err != nil {
		return fmt.Errorf("failed to set total xqore supply: %w", err)
	}

	k.logger.Info("unlocked xQORE",
		"owner", owner.String(),
		"amount", amount.String(),
		"penalty", penaltyAmount.String(),
		"burned", burnAmount.String(),
		"redistributed", redistAmount.String(),
		"returned", returnAmount.String(),
		"height", ctx.BlockHeight(),
	)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"xqore_unlock",
			sdk.NewAttribute("owner", owner.String()),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("penalty", penaltyAmount.String()),
			sdk.NewAttribute("burned", burnAmount.String()),
			sdk.NewAttribute("redistributed", redistAmount.String()),
			sdk.NewAttribute("returned", returnAmount.String()),
		),
	)

	// TODO(v1.1): Implement proportional return based on module account balance
	// to realize the PvP rebase effect. Currently, redistAmount stays in the module
	// account but is not proportionally distributed to remaining holders on unlock.
	// For now, the penalty burn still reduces supply, providing indirect value to holders.
	_ = redistAmount

	return nil
}

// findPenaltyRate walks the exit penalty schedule and returns the penalty rate
// for the highest tier whose MinDuration is <= the elapsed time.
// The schedule is assumed to be sorted ascending by MinDuration.
func (k Keeper) findPenaltyRate(schedule []types.PenaltyTier, elapsed time.Duration) math.LegacyDec {
	rate := math.LegacyZeroDec()
	for _, tier := range schedule {
		if elapsed >= tier.MinDuration {
			rate = tier.PenaltyRate
		} else {
			break
		}
	}
	return rate
}

// --- Genesis ---

// InitGenesis initializes the module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set xqore params: %v", err))
	}
	for _, pos := range gs.Positions {
		if err := k.setPosition(ctx, pos); err != nil {
			panic(fmt.Sprintf("failed to set xqore position: %v", err))
		}
	}
	if err := k.setTotalLocked(ctx, gs.TotalLocked); err != nil {
		panic(fmt.Sprintf("failed to set total locked: %v", err))
	}
	if err := k.setTotalXQORESupply(ctx, gs.TotalXQORE); err != nil {
		panic(fmt.Sprintf("failed to set total xqore supply: %v", err))
	}
}

// ExportGenesis exports the module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	positions := k.GetAllPositions(ctx)
	if len(positions) > types.MaxExportPositions {
		positions = positions[:types.MaxExportPositions]
	}
	return &types.GenesisState{
		Params:      k.GetParams(ctx),
		Positions:   positions,
		TotalLocked: k.GetTotalLocked(ctx),
		TotalXQORE:  k.GetTotalXQORESupply(ctx),
	}
}
