package burn

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

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
type BurnFailedFeeDecorator struct {
	keeper BurnKeeper
}

// NewBurnFailedFeeDecorator creates the ante half of the failed_tx channel.
func NewBurnFailedFeeDecorator(keeper BurnKeeper) BurnFailedFeeDecorator {
	return BurnFailedFeeDecorator{keeper: keeper}
}

func (d BurnFailedFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !simulate && ctx.ExecMode() == sdk.ExecModeFinalize {
		if feeTx, ok := tx.(sdk.FeeTx); ok {
			fee := feeTx.GetFee().AmountOf(types.BondDenom)
			if fee.IsPositive() && d.keeper.RecordTxFee(ctx, fee) {
				ctx = ctx.WithValue(failedFeeKey{}, fee)
			}
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
