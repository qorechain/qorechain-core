package burn

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/qorechain/qorechain-core/x/burn/types"
)

// BurnTxCountDecorator increments the per-block transaction counter
// in the burn module. Used for milestone burn tracking.
type BurnTxCountDecorator struct {
	keeper BurnKeeper
}

// NewBurnTxCountDecorator creates a new BurnTxCountDecorator.
func NewBurnTxCountDecorator(keeper BurnKeeper) BurnTxCountDecorator {
	return BurnTxCountDecorator{keeper: keeper}
}

func (d BurnTxCountDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !simulate {
		d.keeper.IncrementBlockTxCount(ctx)
	} else {
		// A simulation is charged the counter's gas on a branch that is never
		// written, so the estimate matches what delivery uses. Simulation
		// never runs in a block, so no block's gas changes.
		branch, _ := ctx.CacheContext()
		d.keeper.IncrementBlockTxCount(branch)
	}
	return next(ctx, tx, simulate)
}

// failedFeeKey carries, from the ante handler to the post handler of the same
// transaction, the fee the ante handler added to the block's failed-fee
// accumulator. baseapp builds the message and post-handler contexts from the
// context the ante handler returns, so the value travels with the transaction
// and never with another one.
type failedFeeKey struct{}

// BurnFailedFeeDecorator implements the ante half of the failed_tx burn
// channel (QSR-2026-0255). It must run after the fee is deducted.
//
// Nothing used to record which fees came from failed transactions, so the
// failed_tx_burn_rate parameter was never applied. The two halves work with
// baseapp's commit rules: the ante handler's writes are kept whether or not the
// messages succeed, and the post handler's writes are kept only when they do.
// So the ante handler adds every delivered transaction's fee, the post handler
// takes it back out on success, and what is left at EndBlock is exactly the
// fees of failed transactions, out-of-gas included (a panic skips the post
// handler's commit as well).
//
// Only block execution counts: CheckTx, simulation and proposal handling do not
// touch the accumulator. The keeper writes it off the gas meter, so gas_used is
// the same with and without the channel, and it writes nothing at all before
// the channel's protocol feature is enabled.
//
// The amount recorded is the fee the DeductFee decorator actually took, not the
// fee the transaction declares. With a dynamic-fee extension the fee checker
// charges min(base fee + tip, fee cap) * gas, which can be far below the
// declared amount; recording the declared amount would let one cheap failing
// transaction with an inflated declared fee make EndBlock burn
// failed_tx_burn_rate of every other transaction's fees in fee_collector. So
// the decorator is given the same fee checker DeductFee runs and records its
// result. The checker is pure (no store access, no gas).
//
// A fee paid in a non-native denom is converted before DeductFee would run, and
// the transaction the later decorators see still carries the foreign fee, for
// which the checker errors or finds no uqor. Re-deriving the fee from the
// transaction then records nothing (QSR-2026-0624 F-1, QSR-2026-0633). From
// the converted-fee switch the decorator records the uqor the conversion paid
// to the fee collector, which the conversion step hands over in the context.
type BurnFailedFeeDecorator struct {
	keeper     BurnKeeper
	feeChecker ante.TxFeeChecker

	convertedFee        func(ctx sdk.Context) (math.Int, bool)
	convertedFeeEnabled func(ctx sdk.Context) bool
}

// NewBurnFailedFeeDecorator creates the ante half of the failed_tx channel.
// feeChecker must be the checker the chain's DeductFee decorator uses; nil
// means the declared fee is what is deducted (the SDK default checker).
func NewBurnFailedFeeDecorator(keeper BurnKeeper, feeChecker ante.TxFeeChecker) BurnFailedFeeDecorator {
	return BurnFailedFeeDecorator{keeper: keeper, feeChecker: feeChecker}
}

// WithConvertedFee makes the decorator record, for a transaction whose fee was
// converted, the native amount source reports, once enabled answers true.
// enabled is read only for converted transactions and must not touch the gas
// meter. Without it (or below the switch) the fee is re-derived from the
// transaction as before.
func (d BurnFailedFeeDecorator) WithConvertedFee(source func(ctx sdk.Context) (math.Int, bool), enabled func(ctx sdk.Context) bool) BurnFailedFeeDecorator {
	d.convertedFee = source
	d.convertedFeeEnabled = enabled
	return d
}

// deductedFee is the uqor fee DeductFee took from the transaction, or, for a
// converted fee from the switch on, the uqor the conversion paid.
func (d BurnFailedFeeDecorator) deductedFee(ctx sdk.Context, tx sdk.Tx) (math.Int, bool) {
	if d.convertedFee != nil && d.convertedFeeEnabled != nil {
		if paid, ok := d.convertedFee(ctx); ok && d.convertedFeeEnabled(ctx) {
			return paid, true
		}
	}
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return math.Int{}, false
	}
	fee := feeTx.GetFee()
	if d.feeChecker != nil {
		charged, _, err := d.feeChecker(ctx, tx)
		if err != nil {
			return math.Int{}, false
		}
		fee = charged
	}
	return fee.AmountOf(types.BondDenom), true
}

func (d BurnFailedFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !simulate && ctx.ExecMode() == sdk.ExecModeFinalize {
		if fee, ok := d.deductedFee(ctx, tx); ok && fee.IsPositive() && d.keeper.RecordTxFee(ctx, fee) {
			ctx = ctx.WithValue(failedFeeKey{}, fee)
		}
	}
	return next(ctx, tx, simulate)
}

// BurnFailedFeePostDecorator is the post half: on success it takes the fee the
// ante half recorded back out of the accumulator.
type BurnFailedFeePostDecorator struct {
	keeper BurnKeeper
}

// NewBurnFailedFeePostDecorator creates the post half of the failed_tx channel.
func NewBurnFailedFeePostDecorator(keeper BurnKeeper) BurnFailedFeePostDecorator {
	return BurnFailedFeePostDecorator{keeper: keeper}
}

func (d BurnFailedFeePostDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (sdk.Context, error) {
	if success && !simulate {
		if fee, ok := ctx.Value(failedFeeKey{}).(math.Int); ok {
			d.keeper.ReleaseTxFee(ctx, fee)
		}
	}
	return next(ctx, tx, simulate, success)
}
