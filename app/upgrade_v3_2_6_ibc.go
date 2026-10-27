package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
)

// upgradeV3_2_6IBC is the IBC applications' one-time state work at the v3.2.6
// plan height.
//
// The three modules are new to a running chain, and the release handler records
// them in the version map without running their InitGenesis (it never runs
// migrations; see mergeMissingModuleVersions). Of the three, only the rate
// limiter has state it cannot do without: its hour epoch, which its begin
// blocker reads from the switch on. It is started here, at the plan height,
// with no limits, no blacklist and no whitelist, so the switch alone changes no
// transfer. Packet Forward Middleware starts with no packets in flight and
// 08-wasm with no code, which is what their empty prefixes already say.
// Idempotent: a chain that already has the epoch keeps it.
func (app *QoreChainApp) upgradeV3_2_6IBC(ctx sdk.Context) error {
	m := newRateLimitAppModule(app.appCodec, *app.RateLimitKeeper, app.rateLimitStore, nil)
	if m.hasHourEpoch(ctx) {
		return nil
	}
	initRateLimitGenesis(ctx, *app.RateLimitKeeper, *ratelimittypes.DefaultGenesis())
	ctx.Logger().Info("v3.2.6: IBC rate limiter initialised", "hour_epoch_start_height", ctx.BlockHeight())
	return nil
}
