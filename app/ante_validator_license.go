package app

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/x/authz"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	licensemod "github.com/qorechain/qorechain-core/x/license"
	licensetypes "github.com/qorechain/qorechain-core/x/license/types"
)

// MinValidatorSelfBond is the minimum self-bond (in uqor) required to create a
// validator on QoreChain: 100,000 QOR.
var MinValidatorSelfBond = math.NewInt(100_000_000_000)

// ValidatorLicenseDecorator gates MsgCreateValidator on two on-chain rules:
//  1. the operator holds an active validator_operator license (granted by the
//     governance authority after the off-chain dashboard fee), and
//  2. the self-bond is at least MinValidatorSelfBond (100,000 QOR).
//
// Genesis (height 0) is exempt: gentx validators are bootstrapped before any
// license can exist. Existing validators are unaffected — only NEW validator
// creation is gated.
type ValidatorLicenseDecorator struct {
	licenseKeeper licensemod.LicenseKeeper
	stakingKeeper *stakingkeeper.Keeper
}

// NewValidatorLicenseDecorator builds the decorator.
func NewValidatorLicenseDecorator(lk licensemod.LicenseKeeper, sk *stakingkeeper.Keeper) ValidatorLicenseDecorator {
	return ValidatorLicenseDecorator{licenseKeeper: lk, stakingKeeper: sk}
}

// AnteHandle enforces the validator licensing + self-bond rules.
func (d ValidatorLicenseDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Genesis gentxs bypass the ante handler entirely, but guard defensively.
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	return d.checkMsgs(ctx, tx.GetMsgs(), 0, next, tx, simulate)
}

// checkMsgs walks the message tree, descending into authz MsgExec.
//
// WHY THE WALK. The gate used to look only at the transaction's top-level messages.
// authz wraps its payload in Any, so a MsgCreateValidator inside a MsgExec was
// invisible to it, and both rules it enforces were bypassed at once: the
// validator_operator licence and the 100,000 QOR self-bond floor. Granting yourself
// an authz grant costs nothing and needs nobody's permission, so this was not a
// narrow path. The wasm mempool guard in this same package already walks the tree
// for exactly this reason; the licence gate simply had not been given the same
// treatment.
func (d ValidatorLicenseDecorator) checkMsgs(
	ctx sdk.Context, msgs []sdk.Msg, depth int, next sdk.AnteHandler, tx sdk.Tx, simulate bool,
) (sdk.Context, error) {
	if depth > 4 {
		return ctx, errorsmod.Wrap(errortypes.ErrUnauthorized,
			"authz nesting too deep to inspect for validator creation")
	}

	for _, msg := range msgs {
		if exec, ok := msg.(*authz.MsgExec); ok {
			inner, err := exec.GetMessages()
			if err != nil {
				// A payload that will not unpack cannot be inspected, so it cannot
				// be shown to satisfy the gate.
				return ctx, errorsmod.Wrap(errortypes.ErrUnauthorized,
					"authz payload could not be unpacked for inspection")
			}
			if _, err := d.checkMsgs(ctx, inner, depth+1, nil, nil, simulate); err != nil {
				return ctx, err
			}
			continue
		}

		// THE FLOOR IS AN OPERATING THRESHOLD, NOT ONLY AN ADMISSION ONE. It used to
		// be checked exactly once, at creation, so an operator could admit
		// themselves with 100,000 QOR and then undelegate down to 1 uqor while the
		// validator kept signing. A self-bond that only has to exist at the door is
		// a deposit, not a bond.
		if err := d.refuseSelfBondBelowFloor(ctx, msg); err != nil {
			return ctx, err
		}

		createVal, ok := msg.(*stakingtypes.MsgCreateValidator)
		if !ok {
			continue
		}

		// Self-bond floor.
		if createVal.Value.Amount.LT(MinValidatorSelfBond) {
			return ctx, errorsmod.Wrapf(errortypes.ErrInvalidRequest,
				"validator self-bond %s uqor below required minimum %s uqor (100,000 QOR)",
				createVal.Value.Amount, MinValidatorSelfBond)
		}

		// License gate — the operator account (derived from the valoper address)
		// must hold an active validator_operator license.
		valAddr, err := sdk.ValAddressFromBech32(createVal.ValidatorAddress)
		if err != nil {
			return ctx, errorsmod.Wrap(errortypes.ErrInvalidAddress, "invalid validator address")
		}
		operator := sdk.AccAddress(valAddr).String()
		if !d.licenseKeeper.HasActiveLicense(ctx, operator, licensetypes.FeatureValidatorOperator) {
			return ctx, errorsmod.Wrapf(errortypes.ErrUnauthorized,
				"creating a validator requires an active %q license (granted by governance authority)",
				licensetypes.FeatureValidatorOperator)
		}
	}

	if next == nil {
		// Recursive call: nothing to hand on, the caller continues the chain.
		return ctx, nil
	}
	return next(ctx, tx, simulate)
}

// refuseSelfBondBelowFloor refuses an undelegation or redelegation by a validator's
// own operator that would leave the self-bond under MinValidatorSelfBond.
//
// Only the operator's OWN delegation to their OWN validator counts as self-bond, so
// an ordinary delegator moving stake is untouched, and so is an operator moving
// stake they hold on someone else's validator. Genesis validators are not exempt
// here: the floor is the rule for operating, and it did not apply to them only
// because nothing checked it after admission.
func (d ValidatorLicenseDecorator) refuseSelfBondBelowFloor(ctx sdk.Context, msg sdk.Msg) error {
	if d.stakingKeeper == nil {
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

	val, err := d.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return nil
	}
	del, err := d.stakingKeeper.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil
	}
	selfBond := val.TokensFromShares(del.Shares).TruncateInt()
	remaining := selfBond.Sub(leaving.Amount)
	if remaining.LT(MinValidatorSelfBond) {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest,
			"this would leave the validator's self-bond at %s uqor, below the required "+
				"minimum of %s uqor (100,000 QOR); the floor applies for as long as the "+
				"validator operates, not only when it is created",
			remaining, MinValidatorSelfBond)
	}
	return nil
}
