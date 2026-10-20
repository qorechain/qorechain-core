package app

import (
	"bytes"
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	protov2 "google.golang.org/protobuf/proto"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

type pqcAuthzTx struct{ msgs []sdk.Msg }

func (t pqcAuthzTx) GetMsgs() []sdk.Msg                    { return t.msgs }
func (t pqcAuthzTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

// QSR-2026-0521: from the replay-guard switch an account's own PQC key
// messages are refused inside authz MsgExec; below it they pass, as they did.
func TestPQCKeyAuthzLimiter(t *testing.T) {
	ctx := sdk.NewContext(nil, cmtproto.Header{Height: 9}, false, log.NewNopLogger())
	grantee := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	granter := sdk.AccAddress(bytes.Repeat([]byte{2}, 20)).String()
	next := func(c sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return c, nil }

	exec := func(inner sdk.Msg) sdk.Tx {
		m := authz.NewMsgExec(grantee, []sdk.Msg{inner})
		return pqcAuthzTx{msgs: []sdk.Msg{&m}}
	}
	rotate := &pqctypes.MsgRotatePQCKey{Sender: granter}
	nested := authz.NewMsgExec(grantee, []sdk.Msg{rotate})

	cases := []struct {
		name    string
		tx      sdk.Tx
		refused bool
	}{
		{"rotate in exec", exec(rotate), true},
		{"migrate in exec", exec(&pqctypes.MsgMigratePQCKey{Sender: granter}), true},
		{"register in exec", exec(&pqctypes.MsgRegisterPQCKey{Sender: granter}), true},
		{"register v2 in exec", exec(&pqctypes.MsgRegisterPQCKeyV2{Sender: granter}), true},
		{"rotate in nested exec", exec(&nested), true},
		{"bank send in exec", exec(&banktypes.MsgSend{FromAddress: granter}), false},
		{"rotate at top level", pqcAuthzTx{msgs: []sdk.Msg{rotate}}, false},
	}
	for _, on := range []bool{false, true} {
		d := pqcKeyAuthzLimiter(gateFakePQC{enabled: map[string]bool{pqctypes.FeaturePQCRotationReplayGuard: on}})
		for _, c := range cases {
			_, err := d.AnteHandle(ctx, c.tx, false, next)
			if want := on && c.refused; (err != nil) != want {
				t.Errorf("switch=%v %s: err=%v, want refused=%v", on, c.name, err, want)
			}
		}
	}
}
