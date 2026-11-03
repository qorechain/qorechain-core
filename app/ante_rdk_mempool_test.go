package app

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	rdktypes "github.com/qorechain/qorechain-core/x/rdk/types"
)

// Only the leading MsgSubmitBatch messages are checked: anything after another
// message (which may set the rollup's key in the same transaction) is left to
// execution.
func TestRDKMempoolChecks(t *testing.T) {
	b1, b2 := &rdktypes.MsgSubmitBatch{RollupID: "a"}, &rdktypes.MsgSubmitBatch{RollupID: "b"}
	setKey := &rdktypes.MsgSetVerificationKey{RollupID: "a"}
	for _, tc := range []struct {
		name string
		msgs []sdk.Msg
		want []sdk.Msg
	}{
		{"single batch", []sdk.Msg{b1}, []sdk.Msg{b1}},
		{"two batches", []sdk.Msg{b1, b2}, []sdk.Msg{b1, b2}},
		{"key set first", []sdk.Msg{setKey, b1}, nil},
		{"key set between", []sdk.Msg{b1, setKey, b2}, []sdk.Msg{b1}},
		{"other message first", []sdk.Msg{&banktypes.MsgSend{}, b1}, nil},
		{"authz wrapped", []sdk.Msg{&authz.MsgExec{}}, nil},
	} {
		got := rdkMempoolChecks(tc.msgs)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: %d checked, want %d", tc.name, len(got), len(tc.want))
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: message %d differs", tc.name, i)
			}
		}
	}
}

// CheckTx only: DeliverTx and simulation are untouched, a nil check is off,
// and a refusal is the check's own error (rdk code 21 from the keeper).
func TestRDKZKBatchMempoolDecorator(t *testing.T) {
	refuse := func(sdk.Context, sdk.Msg) error { return rdktypes.ErrVerificationKeyMissing }
	tx := &memTx{msgs: []sdk.Msg{&rdktypes.MsgSubmitBatch{RollupID: "a"}}}
	checkTx := sdk.Context{}.WithIsCheckTx(true)
	for _, c := range []struct {
		name     string
		check    func(sdk.Context, sdk.Msg) error
		ctx      sdk.Context
		simulate bool
		admits   bool
	}{
		{"nil check", nil, checkTx, false, true},
		{"CheckTx", refuse, checkTx, false, false},
		{"ReCheckTx", refuse, checkTx.WithIsReCheckTx(true), false, false},
		{"simulate", refuse, checkTx, true, true},
		{"DeliverTx", refuse, sdk.Context{}, false, true},
	} {
		n := 0
		_, err := NewRDKZKBatchMempoolDecorator(c.check).AnteHandle(c.ctx, tx, c.simulate, reached(&n))
		if c.admits && (err != nil || n != 1) {
			t.Errorf("%s: err %v, next reached %d times; want admitted", c.name, err, n)
		}
		if !c.admits && (!errors.Is(err, rdktypes.ErrVerificationKeyMissing) || n != 0) {
			t.Errorf("%s: err %v, next reached %d times; want ErrVerificationKeyMissing", c.name, err, n)
		}
	}
}
