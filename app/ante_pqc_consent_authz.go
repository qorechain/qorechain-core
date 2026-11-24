package app

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"

	aatypes "github.com/qorechain/qorechain-core/x/abstractaccount/types"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	stakingtiertypes "github.com/qorechain/qorechain-core/x/stakingtier/types"
)

// consentAuthzLimitedMsgTypeURLs are the messages no grantee may send for an
// account from FeaturePQCConsentAuthzLimiter:
//   - the EVM window and EIP-7702 consent messages, a statement made with the
//     account's post-quantum key;
//   - the abstract account management messages, which decide who may
//     authorise the account at all;
//   - the staking tier early unlock, which burns part of the delegator's
//     locked stake;
//   - MsgGrant itself, so a grant cannot be used to issue further grants in
//     the granter's name.
func consentAuthzLimitedMsgTypeURLs() []string {
	urls := pqctypes.PQCConsentMsgTypeURLs()
	return append(urls,
		sdk.MsgTypeURL(&aatypes.MsgCreateAbstractAccount{}),
		sdk.MsgTypeURL(&aatypes.MsgRegisterAuthenticator{}),
		sdk.MsgTypeURL(&aatypes.MsgRevokeAuthenticator{}),
		sdk.MsgTypeURL(&aatypes.MsgUpdateSpendingRules{}),
		sdk.MsgTypeURL(&stakingtiertypes.MsgUnlockEarly{}),
		sdk.MsgTypeURL(&authz.MsgGrant{}),
	)
}

// pqcConsentAuthzLimiter refuses the consent and account-management messages
// inside authz from FeaturePQCConsentAuthzLimiter. Below the switch the
// decorator is skipped, so earlier blocks replay unchanged. The handlers of
// the consent messages also refuse a message the account did not sign itself
// (FeaturePQCConsentOuterSigner), which covers group proposals and contracts;
// this layer refuses earlier, before the fee, and stops the grants
// themselves from being created.
func pqcConsentAuthzLimiter(pqc pqcmod.PQCKeeper) featureGatedDecorators {
	return featureGatedDecorators{
		pqc:     pqc,
		feature: pqctypes.FeaturePQCConsentAuthzLimiter,
		inner:   []sdk.AnteDecorator{newConsentAuthzLimiter(consentAuthzLimitedMsgTypeURLs()...)},
	}
}

// consentAuthzLimiter is the upstream authz limiter (cosmos/evm
// ante/cosmos/authz.go) with one difference: a MsgGrant inside a MsgExec is
// itself an inner message, and is refused when MsgGrant is listed. Upstream
// only checks what such a grant authorises, so a grant of MsgGrant already on
// the chain stayed usable.
type consentAuthzLimiter struct {
	disabled map[string]struct{}
}

// consentAuthzMaxNesting is the upstream limiter's bound.
const consentAuthzMaxNesting = 7

func newConsentAuthzLimiter(urls ...string) consentAuthzLimiter {
	m := make(map[string]struct{}, len(urls))
	for _, u := range urls {
		m[u] = struct{}{}
	}
	return consentAuthzLimiter{disabled: m}
}

func (d consentAuthzLimiter) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if err := d.check(tx.GetMsgs(), false, 1); err != nil {
		return ctx, errorsmod.Wrapf(errortypes.ErrUnauthorized, "%s", err.Error())
	}
	return next(ctx, tx, simulate)
}

func (d consentAuthzLimiter) isDisabled(url string) bool {
	_, ok := d.disabled[url]
	return ok
}

func (d consentAuthzLimiter) check(msgs []sdk.Msg, inner bool, level int) error {
	if level >= consentAuthzMaxNesting {
		return fmt.Errorf("found more nested msgs than permitted; got: %d, expected: <%d", level, consentAuthzMaxNesting)
	}
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *authz.MsgExec:
			innerMsgs, err := m.GetMessages()
			if err != nil {
				return err
			}
			level++
			if err := d.check(innerMsgs, true, level); err != nil {
				return err
			}
		case *authz.MsgGrant:
			if url := sdk.MsgTypeURL(m); inner && d.isDisabled(url) {
				return fmt.Errorf("found disabled msg type: %s", url)
			}
			authorization, err := m.GetAuthorization()
			if err != nil {
				return err
			}
			if url := authorization.MsgTypeURL(); d.isDisabled(url) {
				return fmt.Errorf("found disabled msg type: %s", url)
			}
		default:
			if url := sdk.MsgTypeURL(msg); inner && d.isDisabled(url) {
				return fmt.Errorf("found disabled msg type: %s", url)
			}
		}
	}
	return nil
}
