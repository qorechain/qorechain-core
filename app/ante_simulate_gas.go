package app

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// SIMULATION PAYS WHAT DELIVERY PAYS.
//
// Several decorators do metered bookkeeping in block execution and skip it
// when simulating, so a gas estimate came back short and every client had to
// add a fixed margin (diana: a hybrid transaction estimated at 129,362 used
// 164,230). The decorators below are node-local: each acts only when simulate
// is set, which never happens in CheckTx, ProcessProposal or FinalizeBlock, and
// does the skipped work on a branch of the context that is never written back
// but shares its gas meter. So the estimate includes the gas, nothing is stored,
// and no block's gas, state or events change. The PQC counters and the burn
// transaction counter do the same in their own decorators.

// simulateChargedDecorator runs inner as a simulation, after first running it
// as delivery on a discarded branch, for an inner decorator whose delivery-only
// work is metered and does not depend on a valid signature (CosmWasm's
// transaction counter). The branch's own chain ends at once, so only inner's
// own work is charged.
type simulateChargedDecorator struct{ inner sdk.AnteDecorator }

func (d simulateChargedDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		branch, _ := ctx.CacheContext()
		_, _ = d.inner.AnteHandle(branch, tx, false, func(c sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return c, nil })
	}
	return d.inner.AnteHandle(ctx, tx, simulate, next)
}

// simulatedFeeDeduction charges a simulation that carries no fee for the fee
// transfer delivery will make. Wallets estimate with an empty fee (the fee
// depends on the estimate), and the SDK's DeductFee decorator moves no coins
// for an empty fee, so that transfer, the largest part of the gap, was never in
// the estimate. A transfer of one base unit from the fee payer (or the granter)
// to the fee collector on a discarded branch costs the same reads and writes
// (the amount only changes the stored balance by a byte or two);
// a payer who cannot cover it is left for delivery to refuse. A simulation that
// names a fee is deducted for real by DeductFee and needs nothing here.
type simulatedFeeDeduction struct {
	accountKeeper ante.AccountKeeper
	bankKeeper    authtypes.BankKeeper
	denom         string
}

func (d simulatedFeeDeduction) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !simulate {
		return next(ctx, tx, simulate)
	}
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok || !feeTx.GetFee().IsZero() {
		return next(ctx, tx, simulate)
	}
	from := sdk.AccAddress(feeTx.FeePayer())
	if granter := sdk.AccAddress(feeTx.FeeGranter()); !granter.Empty() {
		from = granter
	}
	branch, _ := ctx.CacheContext()
	// DeductFee has already read (and charged) the payer's account and used
	// the grant; only the transfer is missing, so the lookup is off the meter.
	acc := d.accountKeeper.GetAccount(branch.WithGasMeter(storetypes.NewInfiniteGasMeter()), from)
	if acc != nil {
		_ = ante.DeductFees(d.bankKeeper, branch, acc, sdk.NewCoins(sdk.NewCoin(d.denom, math.OneInt())))
	}
	return next(ctx, tx, simulate)
}
