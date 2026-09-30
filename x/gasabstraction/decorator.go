package gasabstraction

import (
	"fmt"

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
	// Genesis transactions (gentxs) are delivered during InitChain at block
	// height 0 and legitimately carry no fee. Exempt them from the fee policy.
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}
	// Simulation (gas estimation) usually carries no fee and is exempt, but a
	// simulation that names a non-native fee is told now what delivery would
	// say, instead of estimating a transaction that can never be delivered
	// (QSR-2026-0294). Simulation never runs in block execution, so this
	// changes no block's result.
	if simulate {
		if err := d.refuseUnconvertibleInSimulation(ctx, tx); err != nil {
			return ctx, err
		}
		return next(ctx, tx, simulate)
	}

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

	// SECURITY: Reject zero fees — minimum fee required
	if fees.IsZero() {
		return ctx, fmt.Errorf("gas abstraction: zero fees not permitted; minimum fee required")
	}

	// If fee is in native denom, pass through to standard fee deduction
	if fees[0].Denom == config.NativeDenom {
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

	// FAIL CLOSED: THERE IS NO CONVERSION.
	//
	// The comment this replaced said the swap "is handled at fee deduction". It is
	// not, anywhere: the conversion rates in the configuration appear only in the
	// type defaults, in genesis validation and in a query that echoes them back.
	// An accepted non-native fee reached the native-only deduction path
	// unconverted, and the declared rates, including a 10:1 one, were never
	// applied. Admitting the transaction here promised a conversion the chain does
	// not perform. Until one exists, an accepted-but-unconvertible denom is
	// refused with a message that says so, rather than passed on to fail, or
	// worse to succeed, somewhere that does not know the rate was supposed to
	// matter.
	return ctx, types.ErrUnsupportedDenom.Wrapf(
		"denom %s is listed as accepted for fees, but fee conversion is not "+
			"implemented on this chain; pay fees in %s", feeDenom, config.NativeDenom)
}

// refuseUnconvertibleInSimulation applies the deliver-path denom checks to a
// simulated transaction whose fee is non-empty and not native. An empty fee
// (the usual estimation request) passes.
func (d GasAbstractionDecorator) refuseUnconvertibleInSimulation(ctx sdk.Context, tx sdk.Tx) error {
	if !d.keeper.IsEnabled(ctx) {
		return nil
	}
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil
	}
	fees := feeTx.GetFee()
	if fees.IsZero() {
		return nil
	}
	config := d.keeper.GetConfig(ctx)
	if fees[0].Denom == config.NativeDenom {
		return nil
	}
	if _, ok := config.Token(fees[0].Denom); !ok {
		return types.ErrUnsupportedDenom.Wrapf("denom %s not accepted for fee payment", fees[0].Denom)
	}
	return types.ErrUnsupportedDenom.Wrapf(
		"denom %s is listed as accepted for fees, but fee conversion is not "+
			"available on this path; pay fees in %s", fees[0].Denom, config.NativeDenom)
}
