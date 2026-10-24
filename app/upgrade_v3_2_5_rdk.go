package app

import sdk "github.com/cosmos/cosmos-sdk/types"

// rdkFinalizeCursorSeeder is implemented by the full build's rdk keeper. The
// public build's stub keeps no rollups and does not implement it, so the step
// is a no-op there.
type rdkFinalizeCursorSeeder interface {
	SeedFinalizeCursors(ctx sdk.Context) (int, error)
}

// upgradeV3_2_5RDK is this area's one-time state step at the v3.2.5 plan
// height. It runs before the release switches are enabled.
//
// From rdk-challenge-settlement optimistic and based rollups are finalized by
// walking a per-rollup cursor (the oldest batch that is not final) instead of
// every batch they ever settled. The step seeds that cursor for every rollup
// that already has batches, so the walk starts where the old finalizer left
// off. Mainnet and testnet have no rollups, so there it writes nothing.
func (app *QoreChainApp) upgradeV3_2_5RDK(ctx sdk.Context) error {
	s, ok := app.RDKKeeper.(rdkFinalizeCursorSeeder)
	if !ok {
		return nil
	}
	n, err := s.SeedFinalizeCursors(ctx)
	if err != nil {
		return err
	}
	ctx.Logger().With("upgrade", "v3.2.5", "area", "rdk").
		Info("rdk finalization cursors seeded", "rollups", n)
	return nil
}
