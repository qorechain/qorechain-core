package app

import (
	"context"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	inflationtypes "github.com/qorechain/qorechain-core/x/inflation/types"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
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

// UpgradeNameV3_1_98 is the coordinated governance upgrade for the release that
// bundles the September security batch with the light-node reward accounting.
// Unlike v3.1.85/86 this one WRITES STATE, so the height matters: the handler
// snapshots the module account balance that accumulated while no node was
// eligible into the reserved pool, and from that block on the EndBlocker
// distributes only what has accrued since the previous run, bounded per node.
//
// The snapshot lives here rather than in the EndBlocker precisely because it
// must key off chain state and not off when a process started. Every node runs
// this handler at the same plan height, replaying nodes included, so the write
// is identical everywhere. Below that height the lightnode module writes
// nothing, which keeps historical replay byte-identical to the old binary.
//
// The name is the release version, so the MsgSoftwareUpgrade plan and the
// binary agree by construction. If the release is renumbered, rename this in
// the same commit: a plan whose name matches no handler halts every validator
// at the upgrade height with "no upgrade handler".
const UpgradeNameV3_1_98 = "v3.1.98"

// UpgradeNameV3_2_0 is the same release under the number it ships as. The
// testnet took this batch as "v3.1.98" and keeps that record; mainnet takes it
// as v3.2.0, which is what the EVM-lane post-quantum requirement makes it: a
// breaking change for every wallet on that lane, not a patch. Both names are
// registered and run the SAME handler, so a node replays the testnet's history
// and executes the mainnet plan with one binary.
//
// A network that already ran the v3.1.98 handler and then takes v3.2.0 (the
// testnet, to pick up the EVM window) executes it twice: every step is
// idempotent by construction - the reward pool returns the established record,
// enabling a feature twice is a no-op, removing a circuit entry that is not
// there is a no-op, and the version map merge adds only what is missing.
const UpgradeNameV3_2_0 = "v3.2.0"

// UpgradeNameV3_2_2 is the coordinated governance upgrade for the release that
// carries the consensus fixes from the 2026-09-29 bug-bounty re-verification
// (see ReleaseV3_2_2ProtocolFeatures). v3.2.1 was node-local and had no plan.
const UpgradeNameV3_2_2 = "v3.2.2"

// UpgradeNameV3_2_3 is the coordinated governance upgrade for the release that
// enables ReleaseV3_2_3ProtocolFeatures.
const UpgradeNameV3_2_3 = "v3.2.3"

// relayerFriendlyTxPerMinute is the per-sender ceiling the release sets, in
// transactions per 30-block window. The module ships a default of 10, which is
// a wallet's ceiling, not a relayer's.
const relayerFriendlyTxPerMinute = 600

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

	// The release handler, registered under both names (see UpgradeNameV3_2_0).
	// It is the ONLY place the reward-pool split is established, so that happens
	// at one height network-wide. Idempotent throughout.
	releaseHandler := func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		pool := app.LightNodeKeeper.EnsureRewardPoolInitialized(sdkCtx)
		app.LightNodeKeeper.Logger().Info("lightnode reward pool established at upgrade",
			"height", sdkCtx.BlockHeight(),
			"reserved", pool.Reserved.String(),
			"outstanding", pool.Outstanding.String(),
		)

		// Fixes that live in patched dependencies switch on here, at the plan
		// height, and never before: the patched code asks the application
		// whether the feature is enabled, and below this height the answer
		// is no, so history replays byte-identically.
		for _, f := range pqctypes.ReleaseV3_2_0ProtocolFeatures {
			if err := app.PQCKeeper.EnableProtocolFeature(sdkCtx, f); err != nil {
				return fromVM, err
			}
		}

		// The staker share that was deposited into the distribution module's
		// balance without backing any claim is recorded here, once, at one
		// height on every node. From this block the share stops being
		// stranded, because the fee split leaves it in the fee collector; what
		// had already piled up moves only by a governance vote, exactly like
		// the light-node reserve.
		stranded := app.BurnKeeper.EnsureStrandedSnapshot(sdkCtx)
		app.BurnKeeper.Logger().Info("stranded staker share established at upgrade",
			"height", sdkCtx.BlockHeight(),
			"reserved", stranded.Reserved.String(),
		)

		// The per-sender transaction limit becomes real in this release, and at
		// its default of 10 per 30 blocks it also throttles the accounts that
		// relay for the network: an IBC relayer delivers one message per packet
		// and would stall at ten. x/ai has no MsgUpdateParams at all, so this
		// value can only be set here or at genesis. 600 per 30 blocks is about
		// twenty transactions per second from one sender, which is far above
		// any legitimate wallet and still bounds a flooder.
		aiCfg := app.AIKeeper.GetConfig(sdkCtx)
		if aiCfg.MaxTxPerMinute < relayerFriendlyTxPerMinute {
			aiCfg.MaxTxPerMinute = relayerFriendlyTxPerMinute
			if err := app.AIKeeper.SetConfig(sdkCtx, aiCfg); err != nil {
				return fromVM, err
			}
		}

		// The EVM lane was closed on the circuit breaker ahead of this release
		// as the interim mitigation for the locked-balance defect the feature
		// above fixes. Reopen it here, in the same block the fix takes effect,
		// so the lane comes back without anyone having to remember to send a
		// reset. It comes back WITH the post-quantum authorisation window
		// enforced on it, enabled among the features above: reopening the lane
		// without that requirement would restore a lane on which a classical
		// key alone spends the full balance of a post-quantum account, which is
		// the exposure QSR-2026-0254 describes. Removing an entry that is not
		// there is a no-op, so a chain that never tripped is unaffected.
		if err := app.CircuitKeeper.DisableList.Remove(sdkCtx, sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{})); err != nil {
			return fromVM, err
		}

		// The stored version map only ever listed the dependency-injected
		// modules; the manually registered ones (evm, feemarket, erc20,
		// precisebank, wasm, ibc and every QoreChain module) were absent, so
		// a later upgrade that calls RunMigrations would have re-initialised
		// them from default genesis. Record every module the binary runs at
		// its current consensus version. Nothing is migrated here.
		return mergeMissingModuleVersions(fromVM, app.ModuleManager.GetVersionMap()), nil
	}
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameV3_1_98, releaseHandler)
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameV3_2_0, releaseHandler)

	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameV3_2_2, app.upgradeV3_2_2)
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameV3_2_3, app.upgradeV3_2_3)
}

// mergeMissingModuleVersions returns fromVM with an entry for every module in
// current that fromVM lacks. Entries fromVM already has are left exactly as
// they are: those carry the version state was written at, and changing them is
// what a migration is for.
func mergeMissingModuleVersions(fromVM, current module.VersionMap) module.VersionMap {
	out := make(module.VersionMap, len(fromVM)+len(current))
	for name, v := range fromVM {
		out[name] = v
	}
	for name, v := range current {
		if _, ok := out[name]; !ok {
			out[name] = v
		}
	}
	return out
}
