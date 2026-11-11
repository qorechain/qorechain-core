package app

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// CosmosTxGasCapDecorator refuses, from the cosmos-tx-gas-cap switch, a
// Cosmos-lane transaction whose declared gas limit is above the bound in the
// crossvm params (max_cosmos_tx_gas, default 50,000,000). With the block gas
// unbounded nothing else limits how much gas one transaction may ask for,
// and with it how much work one fee buys. It runs before the fee is taken, so
// a refused transaction pays nothing. Simulation, which sets no real limit,
// is not affected. Below the switch it does nothing and reads nothing.
type CosmosTxGasCapDecorator struct {
	limit func(sdk.Context) (uint64, bool)
}

// NewCosmosTxGasCapDecorator returns the decorator. limit reports the bound
// and whether the switch is on; nil is off.
func NewCosmosTxGasCapDecorator(limit func(sdk.Context) (uint64, bool)) CosmosTxGasCapDecorator {
	return CosmosTxGasCapDecorator{limit: limit}
}

func (d CosmosTxGasCapDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if d.limit == nil || simulate {
		return next(ctx, tx, simulate)
	}
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}
	limit, on := d.limit(ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()))
	if on && feeTx.GetGas() > limit {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidGasLimit,
			"gas limit %d is above the per-transaction bound %d", feeTx.GetGas(), limit)
	}
	return next(ctx, tx, simulate)
}

// cosmosTxGasCap reads the switch and, when it is on, the bound. Both reads
// are free of gas.
func (app *QoreChainApp) cosmosTxGasCap(ctx sdk.Context) (uint64, bool) {
	if app.PQCKeeper == nil || app.CrossVMKeeper == nil ||
		!app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureCosmosTxGasCap) {
		return 0, false
	}
	return app.CrossVMKeeper.GetParams(ctx).CosmosTxGasLimit(), true
}
