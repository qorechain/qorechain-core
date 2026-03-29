package burn

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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
