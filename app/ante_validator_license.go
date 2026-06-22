package app

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
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
}

// NewValidatorLicenseDecorator builds the decorator.
func NewValidatorLicenseDecorator(lk licensemod.LicenseKeeper) ValidatorLicenseDecorator {
	return ValidatorLicenseDecorator{licenseKeeper: lk}
}

// AnteHandle enforces the validator licensing + self-bond rules.
func (d ValidatorLicenseDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Genesis gentxs bypass the ante handler entirely, but guard defensively.
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
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

	return next(ctx, tx, simulate)
}
