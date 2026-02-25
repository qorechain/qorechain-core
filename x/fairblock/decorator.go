package fairblock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// FairBlockDecorator is the ante handler decorator for FairBlock tIBE.
// In v1.2.0, this is a passthrough stub -- always calls next.
// Future versions will implement threshold IBE decryption.
type FairBlockDecorator struct {
	keeper FairBlockKeeper
}

// NewFairBlockDecorator creates a new FairBlock ante decorator.
func NewFairBlockDecorator(keeper FairBlockKeeper) FairBlockDecorator {
	return FairBlockDecorator{keeper: keeper}
}

// AnteHandle implements sdk.AnteDecorator.
// When FairBlock is disabled (default in v1.2.0), this is a passthrough.
func (d FairBlockDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// v1.2.0: passthrough stub -- tIBE decryption not yet activated
	if !d.keeper.IsEnabled(ctx) {
		return next(ctx, tx, simulate)
	}
	// Future: decrypt encrypted txs using tIBE shares
	return next(ctx, tx, simulate)
}
