package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	gasabstractiontypes "github.com/qorechain/qorechain-core/x/gasabstraction/types"
	reputationtypes "github.com/qorechain/qorechain-core/x/reputation/types"
)

// upgradeV3_2_2AnteParity is this area's one-time state work at the v3.2.2 plan
// height, before its switches (EVM ante parity, gas abstraction fees,
// reputation scoring, rlconsensus policy validation, AI Cosmos screening) turn
// on. Every step is idempotent.
func (app *QoreChainApp) upgradeV3_2_2AnteParity(ctx sdk.Context) error {
	// Reputation: the removal hook deletes a validator's record from the switch
	// on; records of validators removed before it would otherwise stay forever.
	// Bounded by the number of validators the chain has ever had.
	pruned, err := app.ReputationKeeper.PruneRemovedValidators(ctx)
	if err != nil {
		return err
	}
	// Records of validators that still exist but are not bonded are no longer
	// rescored from the switch on; mark them with the height they stop at.
	// Their lifetime counters are left as they are: scoring v2 reads
	// performance from x/slashing, not from them.
	inactive, err := app.ReputationKeeper.MarkInactiveRecords(ctx)
	if err != nil {
		return err
	}
	// Make the rescoring interval explicit in the stored params.
	repParams := app.ReputationKeeper.GetParams(ctx)
	if repParams.EpochBlocks <= 0 {
		repParams.EpochBlocks = reputationtypes.DefaultEpochBlocks
		// As stored: this handler must replay exactly as it ran (SetParams
		// now validates, QSR-2026-0601).
		if err := app.ReputationKeeper.SetParamsAsStored(ctx, repParams); err != nil {
			return err
		}
	}

	// AI screening: the per-height history the old screening kept is read by
	// nothing from the switch on and was never pruned (a handful of entries).
	aiPruned := app.AIKeeper.PruneLegacyTxHistory(ctx)

	// Fee conversion.
	if err := app.migrateGasAbstractionV3_2_2(ctx); err != nil {
		return err
	}

	ctx.Logger().Info("v3.2.2 ante parity area prepared",
		"reputation_records_pruned", pruned,
		"reputation_records_inactive", inactive,
		"ai_legacy_history_pruned", aiPruned,
	)
	return nil
}

// placeholderFeeDenoms are the example denoms the original genesis listed as
// accepted fee tokens. Neither is a real denom (an IBC voucher is ibc/<hash>),
// so neither could ever be paid.
var placeholderFeeDenoms = map[string]bool{"ibc/USDC": true, "ibc/ATOM": true}

// migrateGasAbstractionV3_2_2 drops the placeholder fee tokens, fills the
// conversion fields with their defaults (conversion closed: a zero block cap
// until governance adds a real token and opens it), and makes sure the float's
// module account exists.
func (app *QoreChainApp) migrateGasAbstractionV3_2_2(ctx sdk.Context) error {
	cfg := app.GasAbstractionKeeper.GetConfig(ctx)
	kept := cfg.AcceptedTokens[:0:0]
	for _, t := range cfg.AcceptedTokens {
		if !placeholderFeeDenoms[t.Denom] {
			kept = append(kept, t)
		}
	}
	cfg.AcceptedTokens = kept
	if cfg.PremiumBps == 0 {
		cfg.PremiumBps = gasabstractiontypes.DefaultPremiumBps
	}
	if cfg.MaxTipMultiplier == 0 {
		cfg.MaxTipMultiplier = gasabstractiontypes.DefaultMaxTipMultiplier
	}
	if cfg.FloatMin == "" {
		cfg.FloatMin = "0"
	}
	if cfg.BlockConversionCap == "" {
		cfg.BlockConversionCap = "0"
	}
	if err := app.GasAbstractionKeeper.SetConfig(ctx, cfg); err != nil {
		return err
	}

	// The float is the module account. A module address can only hold a
	// plain account if one was planted there (through the EVM, for example);
	// no key exists for it, so such an account is converted in place, keeping
	// its number and whatever it holds. GetModuleAccount would panic on it.
	addr := authtypes.NewModuleAddress(gasabstractiontypes.ModuleName)
	acc := app.AccountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		app.AccountKeeper.GetModuleAccount(ctx, gasabstractiontypes.ModuleName)
		return nil
	}
	if _, ok := acc.(sdk.ModuleAccountI); ok {
		return nil
	}
	base, ok := acc.(*authtypes.BaseAccount)
	if !ok || base.GetPubKey() != nil {
		// Not something this handler can convert safely. Leave it: the
		// conversion path refuses while the float cannot receive, and the
		// block cap keeps conversion closed until governance opens it.
		ctx.Logger().Error("gas abstraction float address holds a non-module account; conversion stays closed",
			"address", addr.String())
		return nil
	}
	app.AccountKeeper.SetAccount(ctx, authtypes.NewModuleAccount(base, gasabstractiontypes.ModuleName))
	return nil
}
