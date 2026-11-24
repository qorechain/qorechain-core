package app

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	aatypes "github.com/qorechain/qorechain-core/x/abstractaccount/types"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	stakingtiertypes "github.com/qorechain/qorechain-core/x/stakingtier/types"
)

// From the consent limiter switch the EVM window and EIP-7702 consent
// messages, the abstract account management messages, the staking tier early
// unlock and MsgGrant itself are refused inside authz MsgExec (any depth) and
// as the authorization of a MsgGrant; below it they pass, as they did.
func TestPQCConsentAuthzLimiter(t *testing.T) {
	ctx := sdk.NewContext(nil, cmtproto.Header{Height: 9}, false, log.NewNopLogger())
	grantee := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	granterAddr := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	granter := granterAddr.String()
	next := func(c sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return c, nil }

	exec := func(inner ...sdk.Msg) *authz.MsgExec {
		m := authz.NewMsgExec(grantee, inner)
		return &m
	}
	tx := func(msgs ...sdk.Msg) sdk.Tx { return pqcAuthzTx{msgs: msgs} }
	expiry := time.Unix(2_000_000_000, 0)
	grant := func(url string) *authz.MsgGrant {
		g, err := authz.NewMsgGrant(granterAddr, grantee, authz.NewGenericAuthorization(url), &expiry)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}

	limited := []sdk.Msg{
		&pqctypes.MsgOpenEVMWindow{Sender: granter},
		&pqctypes.MsgCloseEVMWindow{Sender: granter},
		&pqctypes.MsgAuthorizeEVMDelegation{Sender: granter},
		&pqctypes.MsgClearEVMDelegation{Sender: granter},
		&aatypes.MsgCreateAbstractAccount{Owner: granter},
		&aatypes.MsgRegisterAuthenticator{Owner: granter},
		&aatypes.MsgRevokeAuthenticator{Owner: granter},
		&aatypes.MsgUpdateSpendingRules{Owner: granter},
		&stakingtiertypes.MsgUnlockEarly{Delegator: granter},
	}

	type tc struct {
		name    string
		tx      sdk.Tx
		refused bool
	}
	var cases []tc
	for _, m := range limited {
		url := sdk.MsgTypeURL(m)
		cases = append(cases,
			tc{url + " in exec", tx(exec(m)), true},
			tc{url + " in nested exec", tx(exec(exec(m))), true},
			tc{url + " granted", tx(grant(url)), true},
			tc{url + " granted inside exec", tx(exec(grant(url))), true},
			tc{url + " at top level", tx(m), false},
		)
	}
	sendURL := sdk.MsgTypeURL(&banktypes.MsgSend{})
	grantURL := sdk.MsgTypeURL(&authz.MsgGrant{})
	cases = append(cases,
		tc{"bank send in exec", tx(exec(&banktypes.MsgSend{FromAddress: granter})), false},
		tc{"grant of bank send", tx(grant(sendURL)), false},
		tc{"grant of MsgGrant", tx(grant(grantURL)), true},
		// A grant of MsgGrant already on the chain cannot be used: the
		// grantee's MsgExec{MsgGrant} is refused whatever it grants.
		tc{"exec of a grant of bank send", tx(exec(grant(sendURL))), true},
		tc{"bank send in exec next to a top-level window", tx(&pqctypes.MsgOpenEVMWindow{Sender: grantee.String()}, exec(&banktypes.MsgSend{FromAddress: granter})), false},
	)

	for _, on := range []bool{false, true} {
		d := pqcConsentAuthzLimiter(gateFakePQC{enabled: map[string]bool{pqctypes.FeaturePQCConsentAuthzLimiter: on}})
		for _, c := range cases {
			_, err := d.AnteHandle(ctx, c.tx, false, next)
			if want := on && c.refused; (err != nil) != want {
				t.Errorf("switch=%v %s: err=%v, want refused=%v", on, c.name, err, want)
			}
		}
	}

	// The nesting bound is the upstream limiter's: 7 levels refuse.
	deep := sdk.Msg(&banktypes.MsgSend{FromAddress: granter})
	for i := 0; i < 7; i++ {
		deep = exec(deep)
	}
	d := pqcConsentAuthzLimiter(gateFakePQC{enabled: map[string]bool{pqctypes.FeaturePQCConsentAuthzLimiter: true}})
	if _, err := d.AnteHandle(ctx, tx(deep), false, next); err == nil {
		t.Error("7 nested MsgExec should be refused")
	}
}
