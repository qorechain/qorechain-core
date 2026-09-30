package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	licensetypes "github.com/qorechain/qorechain-core/x/license/types"
)

// upgradeV3_2_2StakingAdmission reports, before the staking-admission-gate switch
// turns on, every validator the admission rules would not admit today. It only
// logs: nothing is jailed, refused or rewritten, and the stores are only read.
//
// Two findings are possible, and neither is an error:
//   - an operator without an active validator_operator licence. The genesis
//     validators were bootstrapped before any licence existed, so on mainnet
//     this lists them and is expected; a validator created after genesis that
//     shows up here came in through a path the ante did not see;
//   - an operator self-bond under the 100,000 QOR floor, which a slash can
//     cause and which the old ante's per-message view let a multi-message
//     transaction produce.
//
// What to do with either is the owner's call; the gate only governs what
// happens from the switch on.
func (app *QoreChainApp) upgradeV3_2_2StakingAdmission(ctx sdk.Context) error {
	for _, f := range app.stakingAdmissionFindings(ctx) {
		app.Logger().Info("v3.2.2 staking admission scan", "validator", f.validator,
			"moniker", f.moniker, "finding", f.finding, "self_bond_uqor", f.selfBond)
	}
	return nil
}

type stakingAdmissionFinding struct {
	validator, moniker, finding, selfBond string
}

// stakingAdmissionFindings lists validators whose operator lacks the licence or
// whose operator self-bond is under the floor. Read only.
func (app *QoreChainApp) stakingAdmissionFindings(ctx sdk.Context) []stakingAdmissionFinding {
	if app.StakingKeeper == nil {
		return nil
	}
	var out []stakingAdmissionFinding
	_ = app.StakingKeeper.IterateValidators(ctx, func(_ int64, v stakingtypes.ValidatorI) bool {
		val, ok := v.(stakingtypes.Validator)
		if !ok {
			return false
		}
		valAddr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
		if err != nil {
			return false
		}
		operator := sdk.AccAddress(valAddr)
		selfBond := "0"
		if del, err := app.StakingKeeper.GetDelegation(ctx, operator, valAddr); err == nil {
			bond := val.TokensFromShares(del.Shares).TruncateInt()
			selfBond = bond.String()
			if bond.IsPositive() && bond.LT(MinValidatorSelfBond) {
				out = append(out, stakingAdmissionFinding{val.OperatorAddress, val.Description.Moniker, "self-bond below floor", selfBond})
			}
		}
		if app.LicenseKeeper == nil || !app.LicenseKeeper.HasActiveLicense(ctx, operator.String(), licensetypes.FeatureValidatorOperator) {
			out = append(out, stakingAdmissionFinding{val.OperatorAddress, val.Description.Moniker, "operator has no active validator_operator licence", selfBond})
		}
		return false
	})
	return out
}
