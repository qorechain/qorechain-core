package gasabstraction

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// GasAbstractionDecorator checks transaction fees and converts non-native
// denomination fees to their native equivalent for fee deduction.
type GasAbstractionDecorator struct {
	keeper GasAbstractionKeeper
}

// NewGasAbstractionDecorator creates a new gas abstraction ante decorator.
func NewGasAbstractionDecorator(keeper GasAbstractionKeeper) GasAbstractionDecorator {
	return GasAbstractionDecorator{keeper: keeper}
}

// AnteHandle implements sdk.AnteDecorator.
// If gas abstraction is disabled or fee is in native denom, passes through.
// Otherwise, verifies the fee denom is accepted and marks context for conversion.
func (d GasAbstractionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !d.keeper.IsEnabled(ctx) {
		return next(ctx, tx, simulate)
	}

	// Check if fee is in a non-native denom
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}

	config := d.keeper.GetConfig(ctx)
	fees := feeTx.GetFee()

	// If fees are empty or in native denom, pass through
	if fees.IsZero() || fees[0].Denom == config.NativeDenom {
		return next(ctx, tx, simulate)
	}

	// Check if the denom is accepted
	feeDenom := fees[0].Denom
	accepted := false
	for _, token := range config.AcceptedTokens {
		if token.Denom == feeDenom {
			accepted = true
			break
		}
	}
	if !accepted {
		return ctx, types.ErrUnsupportedDenom.Wrapf("denom %s not accepted for fee payment", feeDenom)
	}

	// v1.2.0: Static conversion rates -- actual swap handled at fee deduction
	// Future: integrate with DEX or oracle for dynamic rates
	return next(ctx, tx, simulate)
}
