package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	burntypes "github.com/qorechain/qorechain-core/x/burn/types"
)

// strandedReserveCorrector is the burn keeper's one-time v3.2.3 correction of
// the stranded reserve. It is not part of the public BurnKeeper interface; the
// full build's keeper adapter provides it, the public stub does not.
type strandedReserveCorrector interface {
	CorrectStrandedReserve(ctx sdk.Context) (before, after burntypes.StrandedRewards, changed bool, err error)
}

// upgradeV3_2_3Econ is the economics step of the v3.2.3 release handler.
//
// QSR-2026-0341: the v3.2.0 handler recorded the stranded staker share as the
// distribution balance minus a liability that counted unwithdrawn commission
// twice, so about 3.81M QOR on mainnet that backs no claim was left out of the
// releasable reserve. Here, once, at the plan height, the reserve is
// recomputed with the corrected liability and raised if it was too low. Both
// values are logged. Nothing moves: the coins stay in the distribution module
// until a governance vote releases them.
func (app *QoreChainApp) upgradeV3_2_3Econ(ctx sdk.Context) error {
	c, ok := app.BurnKeeper.(strandedReserveCorrector)
	if !ok {
		ctx.Logger().Info("v3.2.3: burn keeper has no stranded reserve correction, skipped")
		return nil
	}
	before, after, changed, err := c.CorrectStrandedReserve(ctx)
	if err != nil {
		return err
	}
	ctx.Logger().Info("v3.2.3: stranded staker share reserve recomputed",
		"height", ctx.BlockHeight(),
		"old_reserved", before.Reserved.String(),
		"new_reserved", after.Reserved.String(),
		"released", after.Released.String(),
		"changed", changed,
	)
	if changed {
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"burn_stranded_reserve_corrected",
			sdk.NewAttribute("old_reserved", before.Reserved.String()),
			sdk.NewAttribute("new_reserved", after.Reserved.String()),
		))
	}
	return nil
}
