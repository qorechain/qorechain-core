package app

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	stakingtiertypes "github.com/qorechain/qorechain-core/x/stakingtier/types"
)

func undel(val string) *stakingtypes.MsgUndelegate {
	return &stakingtypes.MsgUndelegate{DelegatorAddress: "d", ValidatorAddress: val}
}

func redel(src, dst string) *stakingtypes.MsgBeginRedelegate {
	return &stakingtypes.MsgBeginRedelegate{DelegatorAddress: "d", ValidatorSrcAddress: src, ValidatorDstAddress: dst}
}

// Only the messages that execute against the committed state unchanged are
// checked: leading undelegations and redelegations, each on a validator no
// earlier message touched.
func TestStakingLockMempoolChecks(t *testing.T) {
	a1, a2, b, c := undel("A"), undel("A"), undel("B"), redel("C", "B")
	for _, tc := range []struct {
		name string
		msgs []sdk.Msg
		want []sdk.Msg
	}{
		{"single undelegate", []sdk.Msg{a1}, []sdk.Msg{a1}},
		{"two validators", []sdk.Msg{a1, b}, []sdk.Msg{a1, b}},
		{"second on the same validator", []sdk.Msg{a1, a2}, []sdk.Msg{a1}},
		{"redelegation destination touched", []sdk.Msg{c, b}, []sdk.Msg{c}},
		{"other message first", []sdk.Msg{&banktypes.MsgSend{}, a1}, nil},
		{"other message between", []sdk.Msg{a1, &banktypes.MsgSend{}, b}, []sdk.Msg{a1}},
		{"authz wrapped", []sdk.Msg{&authz.MsgExec{}}, nil},
	} {
		got := stakingLockMempoolChecks(tc.msgs)
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
// and a refusal is the check's own error (stakingtier code 13 from the keeper).
func TestStakingLockMempoolDecorator(t *testing.T) {
	refuse := func(sdk.Context, sdk.Msg) error { return stakingtiertypes.ErrLockedStake }
	tx := &memTx{msgs: []sdk.Msg{undel("A")}}
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
		_, err := NewStakingLockMempoolDecorator(c.check).AnteHandle(c.ctx, tx, c.simulate, reached(&n))
		if c.admits && (err != nil || n != 1) {
			t.Errorf("%s: err %v, next reached %d times; want admitted", c.name, err, n)
		}
		if !c.admits && (!errors.Is(err, stakingtiertypes.ErrLockedStake) || n != 0) {
			t.Errorf("%s: err %v, next reached %d times; want ErrLockedStake", c.name, err, n)
		}
	}
}
