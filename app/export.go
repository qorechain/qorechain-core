package app

import (
	"encoding/json"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ExportAppStateAndValidators exports the state of the application for a genesis file.
func (app *QoreChainApp) ExportAppStateAndValidators(
	forZeroHeight bool,
	jailAllowedAddrs []string,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// The exported genesis starts at the height after the last committed
	// block, where the consensus engine will call InitChain; a zero-height
	// export starts from 0. As the Cosmos SDK simapp does.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	genState, err := app.ModuleManager.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          height,
		ConsensusParams: app.GetConsensusParams(ctx),
	}, nil
}

// prepForZeroHeightGenesis prepares a fresh genesis from a running chain's
// state. It follows the Cosmos SDK v0.53 simapp routine (QSR-2026-0589): every
// commission and delegator reward is paid out first, so only the fractional
// scraps left in each validator's outstanding rewards move to the community
// pool, and the distribution records are then rebuilt from zero through the
// distribution hooks. The earlier version moved the full outstanding rewards
// to the community pool without paying or clearing them, so the exported
// genesis counted every unpaid reward twice and did not conserve the
// distribution module's balance.
//
// Operator tooling only: this runs on a branch of the state when an operator
// asks for a zero-height export and never in block execution.
func (app *QoreChainApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	applyAllowedAddrs := len(jailAllowedAddrs) > 0

	allowedAddrsMap := make(map[string]bool)
	for _, addr := range jailAllowedAddrs {
		if _, err := sdk.ValAddressFromBech32(addr); err != nil {
			panic(err)
		}
		allowedAddrsMap[addr] = true
	}

	/* Fee distribution state. */

	valCodec := app.StakingKeeper.ValidatorAddressCodec()
	operatorBytes := func(val stakingtypes.ValidatorI) sdk.ValAddress {
		bz, err := valCodec.StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		return bz
	}

	// Withdraw all validator commission. A validator with none returns an
	// error, which is expected and ignored, as upstream does.
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, operatorBytes(val))
		return false
	})
	if err != nil {
		panic(err)
	}

	// Withdraw all delegator rewards.
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		panic(err)
	}
	for _, delegation := range dels {
		valAddr, err := valCodec.StringToBytes(delegation.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		delAddr, err := app.AccountKeeper.AddressCodec().StringToBytes(delegation.DelegatorAddress)
		if err != nil {
			panic(err)
		}
		_, _ = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr)
	}

	// Clear validator slash events and historical rewards; both are rebuilt
	// from zero below.
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// Reinitialise every validator: donate the scraps left in its outstanding
	// rewards to the community pool, then AfterValidatorCreated resets its
	// outstanding, current and historical rewards and its commission.
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz := operatorBytes(val)
		scraps, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valBz)
		if err != nil {
			panic(err)
		}
		feePool, err := app.DistrKeeper.FeePool.Get(ctx)
		if err != nil {
			panic(err)
		}
		feePool.CommunityPool = feePool.CommunityPool.Add(scraps...)
		if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
			panic(err)
		}
		if err := app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valBz); err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// Reinitialise every delegation's starting info.
	for _, del := range dels {
		valAddr, err := valCodec.StringToBytes(del.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		delAddr, err := app.AccountKeeper.AddressCodec().StringToBytes(del.DelegatorAddress)
		if err != nil {
			panic(err)
		}
		if err := app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
			panic(err)
		}
		if err := app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
			panic(err)
		}
	}

	ctx = ctx.WithBlockHeight(height)

	/* Staking state. */

	// Reset redelegation and unbonding creation heights.
	err = app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		if err := app.StakingKeeper.SetRedelegation(ctx, red); err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}
	err = app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		if err := app.StakingKeeper.SetUnbondingDelegation(ctx, ubd); err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// Reset unbonding heights, and jail every validator not on the allowed
	// list. Jailing goes through the staking keeper with the validator's
	// consensus address, which also takes it out of the power index; the
	// earlier version passed the operator address as a consensus address.
	vals, err := app.StakingKeeper.GetAllValidators(ctx)
	if err != nil {
		panic(err)
	}
	for _, validator := range vals {
		validator.UnbondingHeight = 0
		if err := app.StakingKeeper.SetValidator(ctx, validator); err != nil {
			panic(err)
		}
		if applyAllowedAddrs && !allowedAddrsMap[validator.GetOperator()] && !validator.IsJailed() {
			consAddr, err := validator.GetConsAddr()
			if err != nil {
				panic(err)
			}
			if err := app.StakingKeeper.Jail(ctx, consAddr); err != nil {
				panic(err)
			}
		}
	}

	if _, err := app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx); err != nil {
		panic(err)
	}

	/* Slashing state. */

	// Reset the start height of every signing info, keyed by consensus
	// address as the slashing module keeps them.
	err = app.SlashingKeeper.IterateValidatorSigningInfos(ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			if err := app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info); err != nil {
				panic(err)
			}
			return false
		},
	)
	if err != nil {
		panic(err)
	}
}
