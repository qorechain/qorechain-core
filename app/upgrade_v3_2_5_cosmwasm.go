package app

import (
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// upgradeV3_2_5Wasm is this area's one-time state step at the v3.2.5 plan
// height. It runs before the release switches are enabled.
//
// From v3.2.5 contracts are deployed through governance: only code governance
// stored is meant to be instantiable by anyone once the instantiate default
// becomes Everybody and the circuit breaker lets instantiation through again.
// Code uploaded by accounts while upload was permissionless (diana holds a few
// hundred such codes, mainnet none) was never reviewed and mostly carries an
// Everybody instantiate permission of its own, so it would come back to life
// with that change. The step closes instantiation of every code not stored by
// the governance authority. Existing contracts are left as they are, and
// governance can reopen a reviewed code with MsgUpdateInstantiateConfig.
func (app *QoreChainApp) upgradeV3_2_5Wasm(ctx sdk.Context) error {
	return closeUnreviewedWasmCodes(ctx, &app.WasmKeeper)
}

// closeUnreviewedWasmCodes sets the instantiate permission of every code whose
// creator is not the wasm authority to Nobody. Codes already at Nobody are left
// untouched, so the step writes and emits nothing on a chain with nothing to
// close.
func closeUnreviewedWasmCodes(ctx sdk.Context, k *wasmkeeper.Keeper) error {
	authority := k.GetAuthority()
	var open []uint64
	k.IterateCodeInfos(ctx, func(codeID uint64, info wasmtypes.CodeInfo) bool {
		if info.Creator != authority && info.InstantiateConfig.Permission != wasmtypes.AccessTypeNobody {
			open = append(open, codeID)
		}
		return false
	})
	gov := wasmkeeper.NewGovPermissionKeeper(k)
	authAddr := sdk.MustAccAddressFromBech32(authority)
	for _, codeID := range open {
		if err := gov.SetAccessConfig(ctx, codeID, authAddr, wasmtypes.AllowNobody); err != nil {
			return fmt.Errorf("close instantiation of wasm code %d: %w", codeID, err)
		}
	}
	return nil
}
