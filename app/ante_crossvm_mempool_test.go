package app

import (
	"bytes"
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/group"

	crossvmtypes "github.com/qorechain/qorechain-core/x/crossvm/types"
)

func crossVMCallWithPayload(n int) *crossvmtypes.MsgCrossVMCall {
	return &crossvmtypes.MsgCrossVMCall{
		Sender:         sdk.AccAddress("sender______________").String(),
		SourceVM:       crossvmtypes.VMTypeEVM,
		TargetVM:       crossvmtypes.VMTypeCosmWasm,
		TargetContract: "target",
		Payload:        bytes.Repeat([]byte{1}, n),
	}
}

// A call whose payload is over max_message_size passed CheckTx, was included
// and failed in the keeper with the fee taken. The mempool now refuses it,
// also inside an authz grant; at or under the limit it is admitted.
func TestCrossVMPayloadMempoolDecorator(t *testing.T) {
	const max = 65536
	limit := func(sdk.Context) uint64 { return max }
	grantee := sdk.AccAddress("grantee_____________")
	wrap := func(m sdk.Msg) sdk.Msg { e := authz.NewMsgExec(grantee, []sdk.Msg{m}); return &e }
	policy := sdk.AccAddress("group_policy________").String()
	groupProp := func(m sdk.Msg) sdk.Msg {
		p, err := group.NewMsgSubmitProposal(policy, []string{policy}, []sdk.Msg{m}, "", group.Exec_EXEC_TRY, "t", "s")
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	checkTx := sdk.Context{}.WithIsCheckTx(true)
	for _, c := range []struct {
		name     string
		limit    func(sdk.Context) uint64
		msgs     []sdk.Msg
		ctx      sdk.Context
		simulate bool
		admits   bool
	}{
		{"at the limit", limit, []sdk.Msg{crossVMCallWithPayload(max)}, checkTx, false, true},
		{"over the limit", limit, []sdk.Msg{crossVMCallWithPayload(max + 1)}, checkTx, false, false},
		{"over, after another message", limit, []sdk.Msg{&banktypes.MsgSend{}, crossVMCallWithPayload(max + 1)}, checkTx, false, false},
		{"over, in authz", limit, []sdk.Msg{wrap(crossVMCallWithPayload(max + 1))}, checkTx, false, false},
		{"over, in nested authz", limit, []sdk.Msg{wrap(wrap(crossVMCallWithPayload(max + 1)))}, checkTx, false, false},
		// A failing group proposal does not fail its submission: left alone.
		{"over, in a group proposal", limit, []sdk.Msg{groupProp(crossVMCallWithPayload(max + 1))}, checkTx, false, true},
		// A params update earlier in the same tx may raise the limit.
		{"over, after a params update", limit, []sdk.Msg{&crossvmtypes.MsgUpdateParams{}, crossVMCallWithPayload(max + 1)}, checkTx, false, true},
		{"over, ReCheckTx", limit, []sdk.Msg{crossVMCallWithPayload(max + 1)}, checkTx.WithIsReCheckTx(true), false, false},
		{"over, simulate", limit, []sdk.Msg{crossVMCallWithPayload(max + 1)}, checkTx, true, true},
		{"over, DeliverTx", limit, []sdk.Msg{crossVMCallWithPayload(max + 1)}, sdk.Context{}, false, true},
		{"over, nil limit", nil, []sdk.Msg{crossVMCallWithPayload(max + 1)}, checkTx, false, true},
		{"over, zero limit", func(sdk.Context) uint64 { return 0 }, []sdk.Msg{crossVMCallWithPayload(max + 1)}, checkTx, false, true},
	} {
		n := 0
		_, err := NewCrossVMPayloadMempoolDecorator(c.limit).AnteHandle(c.ctx, &memTx{msgs: c.msgs}, c.simulate, reached(&n))
		if c.admits && (err != nil || n != 1) {
			t.Errorf("%s: err %v, next reached %d times; want admitted", c.name, err, n)
		}
		if !c.admits && (!errors.Is(err, crossvmtypes.ErrMessageTooLarge) || n != 0) {
			t.Errorf("%s: err %v, next reached %d times; want ErrMessageTooLarge", c.name, err, n)
		}
	}
}
