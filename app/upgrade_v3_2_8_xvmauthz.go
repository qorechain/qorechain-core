package app

import sdk "github.com/cosmos/cosmos-sdk/types"

// crossvmQueueOnBehalfMarker is implemented by the full crossvm keeper.
type crossvmQueueOnBehalfMarker interface {
	MarkQueuedOnBehalfV328(ctx sdk.Context) int
}

// upgradeV3_2_8CrossVMQueueOnBehalf marks every call into the EVM queued
// before the upgrade in an ordinary account's name as sent on its behalf
// (crossvm-on-behalf-evm-window): whether it was the account's own message
// cannot be proven any more, so it runs only if the account's EVM window
// admits it. Calls stay queued with what they prepaid. A chain with an empty
// queue (mainnet, MsgCrossVMCall tripped) writes nothing; the stub keeper of
// a public build has no queue.
func (app *QoreChainApp) upgradeV3_2_8CrossVMQueueOnBehalf(ctx sdk.Context) {
	m, ok := app.CrossVMKeeper.(crossvmQueueOnBehalfMarker)
	if !ok {
		return
	}
	if n := m.MarkQueuedOnBehalfV328(ctx); n > 0 {
		app.Logger().Info("v3.2.8: queued cross-VM calls marked as sent on the account's behalf", "count", n)
	}
}
