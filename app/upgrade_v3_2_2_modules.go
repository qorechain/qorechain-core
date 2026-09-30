package app

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	bridgetypes "github.com/qorechain/qorechain-core/x/bridge/types"
)

// upgradeV3_2_2Modules is this area's one-time state work at the v3.2.2 plan
// height, run before the switches turn on.
//
// Bridge (QSR-2026-0016): from the switch, only a route whose status is exactly
// "active" moves value, and admin messages refuse anything but the three
// lower-case statuses. A status stored verbatim before that (any casing, any
// typo) is normalised here so the new rule meets well-formed data: a known
// status in the wrong case is folded to lower case, anything else becomes
// "paused", which is the fail-safe reading, and an event records every change.
// On both networks today every route is "pending", so this changes nothing
// there; it exists for the chain where it would.
//
// The other module fixes in this area (failed_tx burn, emission cap guard,
// governance reward bound, ecdsa_pubkey binding, multilayer params route) need
// no state preparation: each starts from the state that is already there.
func (app *QoreChainApp) upgradeV3_2_2Modules(ctx sdk.Context) error {
	return normaliseBridgeStatuses(ctx, app.BridgeKeeper)
}

// bridgeChainConfigStore is the part of the bridge keeper the normalisation
// needs.
type bridgeChainConfigStore interface {
	GetAllChainConfigs(ctx sdk.Context) []bridgetypes.ChainConfig
	SetChainConfig(ctx sdk.Context, cc bridgetypes.ChainConfig) error
}

// normaliseBridgeStatuses folds stored route statuses to the three valid
// values. Idempotent: a second run finds nothing to change.
func normaliseBridgeStatuses(ctx sdk.Context, k bridgeChainConfigStore) error {
	if k == nil {
		return nil
	}
	for _, cc := range k.GetAllChainConfigs(ctx) {
		if cc.Status.IsValid() {
			continue
		}
		prev := cc.Status
		next := bridgetypes.BridgeStatus(strings.ToLower(strings.TrimSpace(string(prev))))
		if !next.IsValid() {
			next = bridgetypes.BridgeStatusPaused
		}
		cc.Status = next
		if err := k.SetChainConfig(ctx, cc); err != nil {
			return err
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"bridge_chain_status_normalised",
			sdk.NewAttribute("chain_id", cc.ChainID),
			sdk.NewAttribute("previous_status", string(prev)),
			sdk.NewAttribute("status", string(next)),
		))
		ctx.Logger().Info("v3.2.2: bridge route status normalised",
			"chain_id", cc.ChainID, "previous", string(prev), "status", string(next))
	}
	return nil
}
