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
type BurnFailedFeeDecorator struct {
	keeper     BurnKeeper
	feeChecker ante.TxFeeChecker
}

// NewBurnFailedFeeDecorator creates the ante half of the failed_tx channel.
// feeChecker must be the checker the chain's DeductFee decorator uses; nil
// means the declared fee is what is deducted (the SDK default checker).
func NewBurnFailedFeeDecorator(keeper BurnKeeper, feeChecker ante.TxFeeChecker) BurnFailedFeeDecorator {
	return BurnFailedFeeDecorator{keeper: keeper, feeChecker: feeChecker}
}

// deductedFee is the uqor fee DeductFee took from the transaction.
func (d BurnFailedFeeDecorator) deductedFee(ctx sdk.Context, tx sdk.Tx) (math.Int, bool) {
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
