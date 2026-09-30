package app

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	licensetypes "github.com/qorechain/qorechain-core/x/license/types"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// MinValidatorSelfBond is the minimum self-bond (in uqor) required to create a
// validator on QoreChain: 100,000 QOR.
var MinValidatorSelfBond = math.NewInt(100_000_000_000)

// admissionLicences is the part of the licence keeper the admission rule reads.
type admissionLicences interface {
	HasActiveLicense(ctx sdk.Context, grantee, featureID string) bool
}

// admissionStaking is the part of the staking keeper the admission rule reads.
type admissionStaking interface {
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	GetDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.Delegation, error)
}

// selfBondExit says what the operating floor does with an operator's own
// withdrawal that would take the self-bond to exactly zero.
type selfBondExit bool

const (
	// exitRefused is the v3.2.0 rule: anything that leaves less than the floor
	// is refused, a full exit included. Blocks below the staking-admission-gate
	// switch are replayed with it.
	exitRefused selfBondExit = false
	// exitAllowed lets the operator leave completely. The SDK then jails the
	// validator in Unbond, because a self-delegation of zero is below its
	// MinSelfDelegation, so no validator operates under the floor. Leaving a
	// non-zero remainder under the floor is still refused.
	exitAllowed selfBondExit = true
)

// checkValidatorAdmission is the one statement of the validator admission rules.
// The ante decorator applies it to what a transaction carries, and the staking
// message server applies it to every message it executes, whatever dispatched
// it (a transaction, authz, x/group, a contract, a precompile, governance).
//
//   - MsgCreateValidator: the self-bond is at least MinValidatorSelfBond and the
//     operator account (the valoper address as an account) holds an active
//     validator_operator licence.
//   - MsgUndelegate and MsgBeginRedelegate by a validator's own operator from
//     their own validator: the self-bond left behind is not under the floor
//     (see selfBondExit for a full exit).
//
// Everything else passes. Addresses that do not parse pass too, except for a
// MsgCreateValidator's own address: the staking handler reports malformed input
// with its usual error.
func checkValidatorAdmission(
	ctx sdk.Context, lk admissionLicences, sk admissionStaking, msg sdk.Msg, exit selfBondExit,
) error {
	if err := refuseSelfBondBelowFloor(ctx, sk, msg, exit); err != nil {
		return err
	}

	createVal, ok := msg.(*stakingtypes.MsgCreateValidator)
	if !ok {
		return nil
	}

	// Self-bond floor.
	if createVal.Value.Amount.LT(MinValidatorSelfBond) {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest,
			"validator self-bond %s uqor below required minimum %s uqor (100,000 QOR)",
			createVal.Value.Amount, MinValidatorSelfBond)
	}

	// License gate: the operator account (derived from the valoper address)
	// must hold an active validator_operator license.
	valAddr, err := sdk.ValAddressFromBech32(createVal.ValidatorAddress)
	if err != nil {
		return errorsmod.Wrap(errortypes.ErrInvalidAddress, "invalid validator address")
	}
	operator := sdk.AccAddress(valAddr).String()
	if lk == nil || !lk.HasActiveLicense(ctx, operator, licensetypes.FeatureValidatorOperator) {
		return errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"creating a validator requires an active %q license (granted by governance authority)",
			licensetypes.FeatureValidatorOperator)
	}
	return nil
}

// refuseSelfBondBelowFloor refuses an undelegation or redelegation by a validator's
// own operator that would leave the self-bond under MinValidatorSelfBond.
//
// THE FLOOR IS AN OPERATING THRESHOLD, NOT ONLY AN ADMISSION ONE. It used to be
// checked exactly once, at creation, so an operator could admit themselves with
// 100,000 QOR and then undelegate down to 1 uqor while the validator kept
// signing. A self-bond that only has to exist at the door is a deposit, not a
// bond.
//
// Only the operator's OWN delegation to their OWN validator counts as self-bond, so
// an ordinary delegator moving stake is untouched, and so is an operator moving
// stake they hold on someone else's validator. Genesis validators are not exempt
// here: the floor is the rule for operating, and it did not apply to them only
// because nothing checked it after admission.
func refuseSelfBondBelowFloor(ctx sdk.Context, sk admissionStaking, msg sdk.Msg, exit selfBondExit) error {
	if sk == nil {
		return nil
	}

	var delegator, validator string
	var leaving sdk.Coin
	switch m := msg.(type) {
	case *stakingtypes.MsgUndelegate:
		delegator, validator, leaving = m.DelegatorAddress, m.ValidatorAddress, m.Amount
	case *stakingtypes.MsgBeginRedelegate:
		delegator, validator, leaving = m.DelegatorAddress, m.ValidatorSrcAddress, m.Amount
	default:
		return nil
	}

	valAddr, err := sdk.ValAddressFromBech32(validator)
	if err != nil {
		return nil // the staking handler reports the malformed address
	}
	delAddr, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return nil
	}
	if !delAddr.Equals(sdk.AccAddress(valAddr)) {
		return nil // not the operator's own bond
	}

	val, err := sk.GetValidator(ctx, valAddr)
	if err != nil {
		return nil
	}
	del, err := sk.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil
	}
	selfBond := val.TokensFromShares(del.Shares).TruncateInt()
	remaining := selfBond.Sub(leaving.Amount)

	// A FULL EXIT IS NOT A BREACH OF THE FLOOR. The v3.2.0 rule refused any
	// remainder under the floor, zero included, so an operator could never take
	// out their last 100,000 QOR and wind the validator down, and one slashed
	// under the floor could move nothing at all. From the switch on, leaving
	// nothing is allowed: the SDK jails the validator in the same message, so it
	// does not keep signing without its bond. A remainder that is negative is the
	// staking handler's to refuse (more than the delegation holds).
	if exit == exitAllowed && !remaining.IsPositive() {
		return nil
	}
	if remaining.LT(MinValidatorSelfBond) {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest,
			"this would leave the validator's self-bond at %s uqor, below the required "+
				"minimum of %s uqor (100,000 QOR); the floor applies for as long as the "+
				"validator operates, not only when it is created",
			remaining, MinValidatorSelfBond)
	}
	return nil
}

// stakingAdmissionActive reports whether the admission rules are enforced by the
// staking message server (and the operator's full exit allowed). Read off the
// gas meter, so gas is the same on both sides of the switch.
func (app *QoreChainApp) stakingAdmissionActive(ctx sdk.Context) bool {
	return app.PQCKeeper != nil && app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureStakingAdmissionGate)
}

// stakingAdmissionGate is what the patched staking message server calls for
// CreateValidator, Undelegate and BeginRedelegate (wired in deps_gates.go of
// the full build). Off below the switch; at genesis, gentxs are exempt because
// no licence can exist before the chain does.
func (app *QoreChainApp) stakingAdmissionGate(ctx sdk.Context, msg sdk.Msg) error {
	if !app.stakingAdmissionActive(ctx) || app.executingGenesis(ctx) {
		return nil
	}
	var sk admissionStaking
	if app.StakingKeeper != nil {
		sk = app.StakingKeeper
	}
	return checkValidatorAdmission(ctx, app.LicenseKeeper, sk, msg, exitAllowed)
}

// executingGenesis reports whether the context belongs to InitChain, where the
// genesis transactions run.
//
// NOT ONLY HEIGHT ZERO. InitChain runs at height 0 only when the chain's
// initial_height is 1; with a higher initial_height the SDK gives the genesis
// context that height, so a height-0 test lets a chain started at, say, height
// 1000 refuse its own gentxs. The flag is set by the application's InitChainer
// for exactly as long as InitChain runs.
func (app *QoreChainApp) executingGenesis(ctx sdk.Context) bool {
	return ctx.BlockHeight() == 0 || app.inInitChain
}

// initChainerFlaggingGenesis wraps the runtime InitChainer so executingGenesis can
// tell the genesis transactions apart from a block at the chain's first height.
func (app *QoreChainApp) initChainerFlaggingGenesis(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	app.inInitChain = true
	defer func() { app.inInitChain = false }()
	return app.App.InitChainer(ctx, req)
}
