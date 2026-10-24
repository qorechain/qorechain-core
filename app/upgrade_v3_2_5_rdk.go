package app

import sdk "github.com/cosmos/cosmos-sdk/types"

// upgradeV3_2_5RDK is this area's one-time state step at the v3.2.5 plan
// height. It runs before the release switches are enabled.
func (app *QoreChainApp) upgradeV3_2_5RDK(_ sdk.Context) error { return nil }
