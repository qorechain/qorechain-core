package app

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeNameV3_1_85 is the on-chain name of the v3.1.85 coordinated governance
// upgrade. Mainnet submits a MsgSoftwareUpgrade with this name at a scheduled
// height; every validator halts there and resumes on the v3.1.85 binary, so the
// new authenticator-execution message types (MsgExecuteEVM / MsgExecuteCosmos)
// become decodable network-wide at the same block. No re-genesis.
const UpgradeNameV3_1_85 = "v3.1.85"

// registerUpgradeHandlers wires the named upgrade handlers. Called after the
// keepers are constructed. Registering a handler is harmless until a matching
// MsgSoftwareUpgrade actually executes, so the soak binary can carry it safely.
func (app *QoreChainApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameV3_1_85,
		func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			// v3.1.85 adds the EVM + Native authenticator execution lanes. No
			// module bumped its ConsensusVersion, so there are NO state migrations
			// and no store changes (the aa/authnonce + aa/spend prefixes live in
			// the existing abstractaccount store). The only on-chain effect is to
			// mark the module enabled so the client feature-probe (QoreX §8.13,
			// the abstractaccount Config query) reports authenticator execution
			// active — enforcement itself is binary-driven, not gated on this flag.
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			cfg := app.AbstractAccountKeeper.GetConfig(sdkCtx)
			cfg.Enabled = true
			if err := app.AbstractAccountKeeper.SetConfig(sdkCtx, cfg); err != nil {
				return fromVM, err
			}
			return fromVM, nil
		},
	)
}
