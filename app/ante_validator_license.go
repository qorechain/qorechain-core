package app

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	licensemod "github.com/qorechain/qorechain-core/x/license"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// ValidatorLicenseDecorator applies the validator admission rules
// (checkValidatorAdmission) to what a transaction carries:
//  1. MsgCreateValidator needs an active validator_operator license on the
//     operator (granted by the governance authority after the off-chain
//     dashboard fee) and a self-bond of at least MinValidatorSelfBond
//     (100,000 QOR);
//  2. an operator's own undelegation or redelegation may not leave the
//     self-bond under that floor.
//
// Genesis is exempt: gentx validators are bootstrapped before any license can
// exist.
//
// SINCE v3.2.2 THIS IS NOT THE ENFORCEMENT POINT. From the staking-admission-gate
// switch the staking message server applies the same rule to every message it
// executes, which covers the dispatch paths a transaction walker cannot see
// (x/group proposals, contracts, precompiles, governance). The decorator stays
// so a transaction that carries a refused message fails fast in CheckTx, and so
// blocks below the switch replay exactly as they were executed.
type ValidatorLicenseDecorator struct {
	licenseKeeper admissionLicences
	stakingKeeper admissionStaking
	// features answers whether a protocol feature is on; nil means none is.
	features interface {
		ProtocolFeatureEnabled(ctx sdk.Context, name string) bool
	}
	// executingGenesis reports InitChain; nil falls back to height 0.
	executingGenesis func(sdk.Context) bool
}

// NewValidatorLicenseDecorator builds the decorator. features and
// executingGenesis may be nil (no protocol feature on; genesis is height 0).
func NewValidatorLicenseDecorator(
	lk licensemod.LicenseKeeper, sk *stakingkeeper.Keeper,
	features interface {
		ProtocolFeatureEnabled(ctx sdk.Context, name string) bool
	},
	executingGenesis func(sdk.Context) bool,
) ValidatorLicenseDecorator {
	d := ValidatorLicenseDecorator{features: features, executingGenesis: executingGenesis}
	if lk != nil {
		d.licenseKeeper = lk
	}
	if sk != nil {
		d.stakingKeeper = sk
	}
	return d
}

// AnteHandle enforces the validator licensing + self-bond rules.
func (d ValidatorLicenseDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Genesis gentxs are exempt: no licence can exist before the chain does.
	if ctx.BlockHeight() == 0 || (d.executingGenesis != nil && d.executingGenesis(ctx)) {
		return next(ctx, tx, simulate)
	}

	// Below the switch the v3.2.0 rule, a full exit refused; from it the rule the
	// staking message server applies, so the two never disagree.
	exit := exitRefused
	if d.features != nil && d.features.ProtocolFeatureEnabled(ctx, pqctypes.FeatureStakingAdmissionGate) {
		exit = exitAllowed
	}

	return d.checkMsgs(ctx, tx.GetMsgs(), 0, exit, next, tx, simulate)
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
	ctx sdk.Context, msgs []sdk.Msg, depth int, exit selfBondExit, next sdk.AnteHandler, tx sdk.Tx, simulate bool,
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
			if _, err := d.checkMsgs(ctx, inner, depth+1, exit, nil, nil, simulate); err != nil {
				return ctx, err
			}
			continue
		}

		if err := checkValidatorAdmission(ctx, d.licenseKeeper, d.stakingKeeper, msg, exit); err != nil {
			return ctx, err
		}
	}

	if next == nil {
		// Recursive call: nothing to hand on, the caller continues the chain.
		return ctx, nil
	}
	return next(ctx, tx, simulate)
}
