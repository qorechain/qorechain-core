package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// rdkEscrowBackfiller is implemented by the full build's rdk keeper. The
// public build's stub keeps no rollups and does not implement it, so the step
// is a no-op there.
type rdkEscrowBackfiller interface {
	BackfillEscrowLedger(ctx sdk.Context) (int, error)
}

// upgradeV3_2_3Rdk is the RDK area's one-time state work at the v3.2.3 plan
// height (QSR-2026-0513). Before v3.2.3 CreateRollup credited the per-rollup
// escrow ledger with 0, so every rollup's ledger reads 0 and no withdrawal
// has ever been paid. The step sets the ledger of every rollup that is not
// stopped to its escrow figure (the stake when the record predates the
// creation burn), so FeatureRDKEscrowLedger finds each rollup holding exactly
// what it put in. Mainnet and testnet have no rollups, so there it writes
// nothing.
func (app *QoreChainApp) upgradeV3_2_3Rdk(ctx sdk.Context) error {
	b, ok := app.RDKKeeper.(rdkEscrowBackfiller)
	if !ok {
		return nil
	}
	n, err := b.BackfillEscrowLedger(ctx)
	if err != nil {
		return err
	}
	ctx.Logger().With("upgrade", "v3.2.3", "area", "rdk").
		Info("rdk escrow ledger backfilled", "rollups", n)
	return nil
}
