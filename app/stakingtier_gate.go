package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// stakingTiersActive reports whether x/stakingtier is in force. Read off the
// gas meter, so gas is the same on both sides of the switch.
func (app *QoreChainApp) stakingTiersActive(ctx sdk.Context) bool {
	return app.PQCKeeper != nil && app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureStakingTiers)
}

// stakingTierLockGate is what the patched staking message server runs, after
// the admission rules, for Undelegate and BeginRedelegate (wired in
// deps_gates.go of the full build): locked stake cannot leave its delegation.
// It sits in the message server for the same reason the self-bond floor does,
// so it holds for every dispatch path (a transaction, authz MsgExec, x/group,
// a contract, a precompile, governance). Off below the switch, without
// touching the gas meter.
func (app *QoreChainApp) stakingTierLockGate(ctx sdk.Context, msg sdk.Msg) error {
	if app.StakingTierKeeper == nil || !app.stakingTiersActive(ctx) {
		return nil
	}
	return app.StakingTierKeeper.CheckStakingMsg(ctx, msg)
}
