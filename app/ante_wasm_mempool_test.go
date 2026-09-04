package app

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// A plain transfer must pass: the guard has to be invisible to normal traffic.
func TestWasmMempoolGuard_AllowsOrdinaryMsg(t *testing.T) {
	err := rejectBlockedWasmMsgs([]sdk.Msg{&banktypes.MsgSend{}}, 0)
	if err != nil {
		t.Fatalf("ordinary MsgSend was rejected: %v", err)
	}
}

func TestWasmMempoolGuard_BlocksEachWasmEntryPoint(t *testing.T) {
	for _, msg := range []sdk.Msg{
		&wasmtypes.MsgStoreCode{},
		&wasmtypes.MsgInstantiateContract{},
		&wasmtypes.MsgInstantiateContract2{},
	} {
		if err := rejectBlockedWasmMsgs([]sdk.Msg{msg}, 0); err == nil {
			t.Errorf("%s was admitted", sdk.MsgTypeURL(msg))
		}
	}
}

// The obvious bypass: hide the upload inside an authz grant execution.
func TestWasmMempoolGuard_BlocksAuthzWrappedUpload(t *testing.T) {
	exec := authz.NewMsgExec(sdk.AccAddress("grantee_____________"), []sdk.Msg{
		&wasmtypes.MsgStoreCode{},
	})
	if err := rejectBlockedWasmMsgs([]sdk.Msg{&exec}, 0); err == nil {
		t.Fatal("authz-wrapped MsgStoreCode was admitted")
	}
}

// Nesting MsgExec inside MsgExec must not launder the payload either.
func TestWasmMempoolGuard_BlocksDoublyNestedUpload(t *testing.T) {
	grantee := sdk.AccAddress("grantee_____________")
	inner := authz.NewMsgExec(grantee, []sdk.Msg{&wasmtypes.MsgStoreCode{}})
	outer := authz.NewMsgExec(grantee, []sdk.Msg{&inner})
	if err := rejectBlockedWasmMsgs([]sdk.Msg{&outer}, 0); err == nil {
		t.Fatal("doubly nested MsgStoreCode was admitted")
	}
}

// An authz wrapper carrying only benign messages must still pass.
func TestWasmMempoolGuard_AllowsBenignAuthz(t *testing.T) {
	exec := authz.NewMsgExec(sdk.AccAddress("grantee_____________"), []sdk.Msg{
		&banktypes.MsgSend{},
	})
	if err := rejectBlockedWasmMsgs([]sdk.Msg{&exec}, 0); err != nil {
		t.Fatalf("benign authz payload was rejected: %v", err)
	}
}

// Recursion must terminate rather than let a deep chain exhaust the node.
func TestWasmMempoolGuard_RejectsExcessiveNesting(t *testing.T) {
	grantee := sdk.AccAddress("grantee_____________")
	msg := authz.NewMsgExec(grantee, []sdk.Msg{&banktypes.MsgSend{}})
	for i := 0; i < 6; i++ {
		msg = authz.NewMsgExec(grantee, []sdk.Msg{&msg})
	}
	if err := rejectBlockedWasmMsgs([]sdk.Msg{&msg}, 0); err == nil {
		t.Fatal("unbounded authz nesting was admitted")
	}
}
