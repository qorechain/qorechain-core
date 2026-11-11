package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// EventTypeConsentGrantRemoved is emitted by the v3.2.8 release handler for
// each authz grant it removes (see upgradeV3_2_8RemoveConsentGrants).
const EventTypeConsentGrantRemoved = "pqc_consent_grant_removed"

// upgradeV3_2_8RemoveConsentGrants deletes, once at the v3.2.8 plan height,
// every authz grant whose authorization is for a message the consent authz
// limiter refuses (consentAuthzLimitedMsgTypeURLs: the EVM window and
// EIP-7702 consent messages, the abstract account management messages, the
// staking tier early unlock, MsgGrant) or for a post-quantum key management
// message. From v3.2.8 such a grant can no longer be created or used, so it
// authorises nothing; removing it means nothing granted before the release
// survives it, whichever later rule would have had to stop it.
//
// The grants are collected in store order first and deleted afterwards, so
// the iteration never sees its own deletions and every node removes the same
// grants in the same order. Each deletion emits the authz module's own revoke
// event and EventTypeConsentGrantRemoved.
func (app *QoreChainApp) upgradeV3_2_8RemoveConsentGrants(ctx sdk.Context) error {
	limited := make(map[string]struct{})
	for _, u := range consentAuthzLimitedMsgTypeURLs() {
		limited[u] = struct{}{}
	}
	for _, u := range pqctypes.PQCKeyManagementMsgTypeURLs() {
		limited[u] = struct{}{}
	}

	type grantRef struct {
		granter, grantee sdk.AccAddress
		msgType          string
	}
	var remove []grantRef
	app.AuthzKeeper.IterateGrants(ctx, func(granter, grantee sdk.AccAddress, grant authz.Grant) bool {
		auth, err := grant.GetAuthorization()
		if err != nil || auth == nil {
			return false
		}
		if _, ok := limited[auth.MsgTypeURL()]; ok {
			remove = append(remove, grantRef{granter: granter, grantee: grantee, msgType: auth.MsgTypeURL()})
		}
		return false
	})

	for _, g := range remove {
		if err := app.AuthzKeeper.DeleteGrant(ctx, g.grantee, g.granter, g.msgType); err != nil {
			return err
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			EventTypeConsentGrantRemoved,
			sdk.NewAttribute("granter", g.granter.String()),
			sdk.NewAttribute("grantee", g.grantee.String()),
			sdk.NewAttribute("msg_type_url", g.msgType),
			sdk.NewAttribute("reason", "v3.2.8 consent messages are sent by the account itself"),
		))
	}
	return nil
}
