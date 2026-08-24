package app

import (
	"context"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	inflationtypes "github.com/qorechain/qorechain-core/x/inflation/types"
)

// UpgradeNameV3_1_85 is the on-chain name of the v3.1.85 coordinated governance
// upgrade. Mainnet submits a MsgSoftwareUpgrade with this name at a scheduled
// height; every validator halts there and resumes on the v3.1.85 binary, so the
// new authenticator-execution message types (MsgExecuteEVM / MsgExecuteCosmos)
// become decodable network-wide at the same block. No re-genesis.
const UpgradeNameV3_1_85 = "v3.1.85"

// UpgradeNameV3_1_86 is the coordinated governance upgrade that supersedes
// v3.1.85 for the mainnet push. It carries the v3.1.85 authenticator-execution
// lanes AND the jail-lock fix: MetaMask (EIP-191) authenticator verification, and
// the ante escape-hatch that exempts MsgUnjail so a validator whose operator
// account has no PQC key can still recover. All effects are binary/ante-driven;
// no module ConsensusVersion bump, so no state migration. This is the name the
// mainnet MsgSoftwareUpgrade should use (not v3.1.85).
const UpgradeNameV3_1_86 = "v3.1.86"

// UpgradeNameV3_1_94 switches x/inflation from the decaying-schedule emission to
// a fixed per-epoch amount with a hard cumulative ceiling.
//
// Why this is an upgrade handler and not a parameter vote: until this release
// x/inflation had no authority and no MsgUpdateParams at all, so there was no
// on-chain way to change these values. This release adds that path AND uses it
// once, so that every later change goes through governance instead.
//
// The numbers, against emission measured on mainnet over 20,000 blocks
// (36.6568 QOR/block into distribution, 1,364,210 QOR/day):
//
//	FixedEpochEmission  16,239 QOR/epoch  -> ~13,990 QOR/day, ~97x less
//	MaxTotalEmission    114,285,714 QOR   -> a ceiling, not a target
//
// The cap is what makes an allocation real. A per-epoch rate alone drifts with
// block time: epochs are 43,200 blocks, which at the measured 2.3216 s/block is
// 27.9 hours, not the 24 the schedule assumed. The cumulative ceiling holds
// regardless of that drift.
const UpgradeNameV3_1_94 = "v3.1.94"

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

	// v3.1.86 — same shape (no migrations; ensure the authenticator config is
	// enabled). The behavioural changes (EIP-191 authenticator verify, MsgUnjail
	// ante exemption) are entirely binary/ante-level. This is the mainnet target.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameV3_1_86,
		func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			cfg := app.AbstractAccountKeeper.GetConfig(sdkCtx)
			cfg.Enabled = true
			if err := app.AbstractAccountKeeper.SetConfig(sdkCtx, cfg); err != nil {
				return fromVM, err
			}
			return fromVM, nil
		},
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameV3_1_94,
		func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)

			// Read-modify-write rather than construct: the schedule, epoch length
			// and enabled flag stay exactly as they are on chain. Only the three
			// emission fields change, so nothing else can be altered by accident.
			params := app.InflationKeeper.GetParams(sdkCtx)
			params.EmissionMode = inflationtypes.EmissionModeFixed
			params.FixedEpochEmission = inflationtypes.OptionalInt(
				math.NewInt(16_239_000_000)) // 16,239 QOR
			params.MaxTotalEmission = inflationtypes.OptionalInt(
				math.NewInt(114_285_714_000_000)) // 114,285,714 QOR

			// Fail the upgrade rather than write params the module would reject.
			// A handler that returns an error halts the chain at the upgrade
			// height, which is loud and recoverable; silently storing invalid
			// params is neither.
			if err := params.Validate(); err != nil {
				return fromVM, err
			}
			if err := app.InflationKeeper.SetParams(sdkCtx, params); err != nil {
				return fromVM, err
			}

			app.InflationKeeper.Logger().Info("inflation switched to fixed emission",
				"per_epoch", params.FixedEmission().String(),
				"cumulative_cap", params.EmissionCap().String(),
				"height", sdkCtx.BlockHeight(),
			)
			return fromVM, nil
		},
	)
}
