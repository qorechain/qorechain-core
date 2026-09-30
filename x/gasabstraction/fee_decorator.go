package gasabstraction

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// Fee conversion (v3.2.2, QSR-2026-0097, -0158, -0294, -0116).
//
// The documented feature, paying fees in a non-native token, never worked: the
// min-gas-price check refused any fee that was not uqor before the gas
// abstraction decorator ran, nothing converted anything, and the refusal was
// skipped in simulation, so estimation passed and delivery failed.
//
// With the switch on, a fee paid in exactly one non-native denom goes through
// conversion instead of the native fee path. The payer's tokens go to the gas
// abstraction module (the float), and the float pays their value in uqor to the
// fee collector. Everything downstream of the fee collector (the burn split,
// distribution, the fee market) therefore still only ever sees uqor, and the
// native minimum-fee and dynamic-fee rules are applied to that uqor value, so a
// converted fee is held to exactly the rules a native one is. The value is the
// token amount times a rate from a governance-set static rate that expires, or
// the x/amm pool's time-weighted price guarded by depth and deviation from spot,
// less a premium that pays for the float's price risk. Conversion is bounded
// per transaction (a multiple of the minimum fee), per block (a cap) and by a
// balance the float always keeps. Every refusal happens in simulation too.
//
// A fee in the native denom, or in several denoms, takes the native path,
// unchanged. Below the switch everything takes the native path.

// ConversionBankKeeper is an alias of types.ConversionBankKeeper.
type ConversionBankKeeper = types.ConversionBankKeeper

// ConversionAccountKeeper is what conversion needs from x/auth.
type ConversionAccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// ConversionFeegrantKeeper is what conversion needs from x/feegrant.
type ConversionFeegrantKeeper interface {
	UseGrantedFees(ctx context.Context, granter, grantee sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) error
}

// FeeRouterOptions configures NewFeeRouterDecorator.
type FeeRouterOptions struct {
	// Enabled is the protocol-feature switch, read per transaction.
	Enabled func(ctx sdk.Context) bool
	// Native is the native fee path exactly as it ran before conversion
	// existed; it is run unchanged for every transaction that is not
	// converted.
	Native []sdk.AnteDecorator

	Keeper         GasAbstractionKeeper
	BankKeeper     ConversionBankKeeper
	AccountKeeper  ConversionAccountKeeper
	FeegrantKeeper ConversionFeegrantKeeper
	Pools          PoolReader

	// TxSizeGas is the decorator that charges gas for the transaction size,
	// the same one the native path runs.
	TxSizeGas sdk.AnteDecorator
	// MinGasPrice is the native minimum gas price check, applied to the
	// converted value.
	MinGasPrice sdk.AnteDecorator
	// FeeChecker is the native dynamic-fee checker, applied to the converted
	// value; it returns the fee to charge and the priority.
	FeeChecker authante.TxFeeChecker
	// MinFeePerGas returns the minimum native fee per unit of gas at this
	// height (the larger of the base fee and the minimum gas price), for the
	// per-transaction bound.
	MinFeePerGas func(ctx sdk.Context) math.LegacyDec
}

// FeeRouterDecorator sends each transaction down the native fee path or the
// conversion path. See the note at the top of this file.
type FeeRouterDecorator struct{ o FeeRouterOptions }

// NewFeeRouterDecorator returns the decorator.
func NewFeeRouterDecorator(o FeeRouterOptions) FeeRouterDecorator { return FeeRouterDecorator{o: o} }

func (d FeeRouterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if d.o.Enabled == nil || !d.o.Enabled(ctx) || ctx.BlockHeight() == 0 {
		return runChain(ctx, tx, simulate, d.o.Native, next)
	}
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return runChain(ctx, tx, simulate, d.o.Native, next)
	}
	// Routing reads the config off the gas meter: a transaction that takes
	// the native path must cost exactly what it cost before this router.
	cfg := d.o.Keeper.GetConfig(ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()))
	fee := feeTx.GetFee()
	if len(fee) != 1 || fee[0].Denom == cfg.NativeDenom {
		return runChain(ctx, tx, simulate, d.o.Native, next)
	}
	return runChain(ctx, tx, simulate, []sdk.AnteDecorator{
		d.o.TxSizeGas,
		conversionDecorator{o: d.o, cfg: cfg},
	}, next)
}

// runChain runs decs in order and then next.
func runChain(ctx sdk.Context, tx sdk.Tx, simulate bool, decs []sdk.AnteDecorator, next sdk.AnteHandler) (sdk.Context, error) {
	h := next
	for i := len(decs) - 1; i >= 0; i-- {
		dec, n := decs[i], h
		h = func(c sdk.Context, t sdk.Tx, s bool) (sdk.Context, error) { return dec.AnteHandle(c, t, s, n) }
	}
	return h(ctx, tx, simulate)
}

// convertedFeeTx presents tx to the native fee checks with its fee replaced by
// the converted native value.
type convertedFeeTx struct {
	sdk.FeeTx
	fee sdk.Coins
}

func (t convertedFeeTx) GetFee() sdk.Coins { return t.fee }

func (t convertedFeeTx) GetExtensionOptions() []*codectypes.Any {
	if h, ok := t.FeeTx.(authante.HasExtensionOptionsTx); ok {
		return h.GetExtensionOptions()
	}
	return nil
}

func (t convertedFeeTx) GetNonCriticalExtensionOptions() []*codectypes.Any {
	if h, ok := t.FeeTx.(authante.HasExtensionOptionsTx); ok {
		return h.GetNonCriticalExtensionOptions()
	}
	return nil
}

type conversionDecorator struct {
	o   FeeRouterOptions
	cfg types.GasAbstractionConfig
}

// maxFeeBits bounds a converted fee amount so the decimal arithmetic below
// cannot overflow.
const maxFeeBits = 128

func (d conversionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx := tx.(sdk.FeeTx) // checked by the router
	paid := feeTx.GetFee()[0]
	cfg := d.cfg

	if !cfg.Enabled {
		return ctx, types.ErrGasAbstractionOff.Wrapf("pay fees in %s", cfg.NativeDenom)
	}
	token, ok := cfg.Token(paid.Denom)
	if !ok {
		return ctx, types.ErrUnsupportedDenom.Wrapf("denom %s not accepted for fee payment; pay fees in %s", paid.Denom, cfg.NativeDenom)
	}
	if !paid.Amount.IsPositive() {
		return ctx, types.ErrInsufficientFee.Wrap("zero fees not permitted; minimum fee required")
	}
	if paid.Amount.BigInt().BitLen() > maxFeeBits {
		return ctx, types.ErrConversionFailed.Wrap("fee amount too large")
	}
	rate, err := ResolveRate(ctx, cfg, token, d.o.Pools)
	if err != nil {
		return ctx, err
	}
	value := NativeValue(paid.Amount, rate, cfg.PremiumBps)
	if !value.IsPositive() {
		return ctx, types.ErrInsufficientFee.Wrapf("%s is worth nothing in %s at the current rate", paid, cfg.NativeDenom)
	}

	// The native rules, applied to the native value.
	wrapped := convertedFeeTx{FeeTx: feeTx, fee: sdk.NewCoins(sdk.NewCoin(cfg.NativeDenom, value))}
	if _, err := d.o.MinGasPrice.AnteHandle(ctx, wrapped, simulate, func(c sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return c, nil }); err != nil {
		return ctx, err
	}
	charge := wrapped.fee
	priority := int64(0)
	if !simulate {
		charge, priority, err = d.o.FeeChecker(ctx, wrapped)
		if err != nil {
			return ctx, err
		}
	}
	payout := charge.AmountOf(cfg.NativeDenom)
	if !payout.IsPositive() {
		return ctx, types.ErrInsufficientFee.Wrap("converted fee is zero")
	}

	// What the payer gives up: the whole fee, or, when the dynamic-fee rules
	// charge less than the full value, the same share of it (rounded up).
	paidCharged := paid.Amount
	if payout.LT(value) {
		paidCharged = math.LegacyNewDecFromInt(paid.Amount).MulInt(payout).QuoInt(value).Ceil().TruncateInt()
	}

	// Bounds on what the float pays.
	if cfg.MaxTipMultiplier > 0 && d.o.MinFeePerGas != nil {
		ceiling := d.o.MinFeePerGas(ctx).MulInt64(int64(cfg.MaxTipMultiplier)).
			MulInt(math.NewIntFromUint64(feeTx.GetGas())).Ceil().TruncateInt()
		if ceiling.IsPositive() && payout.GT(ceiling) {
			return ctx, types.ErrConversionFailed.Wrapf(
				"converted fee %s%s is above %dx the minimum fee for this gas (%s%s); lower the fee",
				payout, cfg.NativeDenom, cfg.MaxTipMultiplier, ceiling, cfg.NativeDenom)
		}
	}
	blockCap := cfg.BlockCap()
	if !blockCap.IsPositive() {
		return ctx, types.ErrConversionFailed.Wrapf("fee conversion is not open on this chain; pay fees in %s", cfg.NativeDenom)
	}
	used := d.o.Keeper.BlockConversionUsed(ctx)
	if used.Add(payout).GT(blockCap) {
		return ctx, types.ErrConversionFailed.Wrapf("this block's fee conversion capacity is used up; retry, or pay fees in %s", cfg.NativeDenom)
	}
	float := d.o.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.ModuleName), cfg.NativeDenom).Amount
	if float.Sub(payout).LT(cfg.FloatMinimum()) {
		return ctx, types.ErrConversionFailed.Wrapf("the fee conversion float is too low; pay fees in %s", cfg.NativeDenom)
	}

	// Who pays: the fee payer, or its granter.
	payer := sdk.AccAddress(feeTx.FeePayer())
	deductFrom := payer
	foreign := sdk.NewCoins(sdk.NewCoin(paid.Denom, paidCharged))
	if granter := sdk.AccAddress(feeTx.FeeGranter()); len(granter) > 0 && !granter.Equals(payer) {
		if d.o.FeegrantKeeper == nil {
			return ctx, errortypes.ErrInvalidRequest.Wrap("fee grants are not enabled")
		}
		if err := d.o.FeegrantKeeper.UseGrantedFees(ctx, granter, payer, foreign, tx.GetMsgs()); err != nil {
			return ctx, errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", granter, payer)
		}
		deductFrom = granter
	}
	if d.o.AccountKeeper.GetAccount(ctx, deductFrom) == nil {
		return ctx, errortypes.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFrom)
	}
	if err := d.o.BankKeeper.SendCoinsFromAccountToModule(ctx, deductFrom, types.ModuleName, foreign); err != nil {
		return ctx, errorsmod.Wrap(errortypes.ErrInsufficientFunds, err.Error())
	}
	if err := d.o.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName,
		sdk.NewCoins(sdk.NewCoin(cfg.NativeDenom, payout))); err != nil {
		return ctx, errorsmod.Wrap(types.ErrConversionFailed, err.Error())
	}
	d.o.Keeper.AddBlockConversionUsed(ctx, payout)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, foreign.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFrom.String()),
		),
		sdk.NewEvent(types.EventTypeFeeConverted,
			sdk.NewAttribute(types.AttributeKeyPaid, foreign.String()),
			sdk.NewAttribute(types.AttributeKeyNative, fmt.Sprintf("%s%s", payout, cfg.NativeDenom)),
			sdk.NewAttribute(types.AttributeKeyRate, rate.String()),
			sdk.NewAttribute(types.AttributeKeySource, token.PriceSource),
		),
	})
	return next(ctx.WithPriority(priority), tx, simulate)
}
