package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
)

// featureGatedDecorators runs its decorators only once a protocol feature is
// enabled; below the switch it is a straight pass to the next decorator, so the
// chain it sits in behaves exactly as it did without it. The feature read is
// off the gas meter.
type featureGatedDecorators struct {
	pqc     pqcmod.PQCKeeper
	feature string
	inner   []sdk.AnteDecorator
}

func (d featureGatedDecorators) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !d.pqc.ProtocolFeatureEnabled(ctx, d.feature) {
		return next(ctx, tx, simulate)
	}
	return runDecorators(ctx, tx, simulate, d.inner, next)
}

// runDecorators runs decs in order and then next, as sdk.ChainAnteDecorators
// would with next as the terminator.
func runDecorators(ctx sdk.Context, tx sdk.Tx, simulate bool, decs []sdk.AnteDecorator, next sdk.AnteHandler) (sdk.Context, error) {
	h := next
	for i := len(decs) - 1; i >= 0; i-- {
		d, n := decs[i], h
		h = func(c sdk.Context, t sdk.Tx, s bool) (sdk.Context, error) { return d.AnteHandle(c, t, s, n) }
	}
	return h(ctx, tx, simulate)
}
