package app

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/group"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
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
		&wasmtypes.MsgStoreAndInstantiateContract{},
		&wasmtypes.MsgStoreAndMigrateContract{},
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

// From FeatureWasmGovernedDeploy the instantiate messages are admitted, also
// through authz, and every upload message is still refused.
func TestWasmMempoolGuard_GovernedDeployAdmitsInstantiateOnly(t *testing.T) {
	grantee := sdk.AccAddress("grantee_____________")
	for _, msg := range []sdk.Msg{&wasmtypes.MsgInstantiateContract{}, &wasmtypes.MsgInstantiateContract2{}} {
		if err := rejectWasmMsgs([]sdk.Msg{msg}, 0, true); err != nil {
			t.Errorf("%s was refused with the feature on: %v", sdk.MsgTypeURL(msg), err)
		}
		exec := authz.NewMsgExec(grantee, []sdk.Msg{msg})
		if err := rejectWasmMsgs([]sdk.Msg{&exec}, 0, true); err != nil {
			t.Errorf("authz-wrapped %s was refused with the feature on: %v", sdk.MsgTypeURL(msg), err)
		}
	}
	for _, msg := range []sdk.Msg{
		&wasmtypes.MsgStoreCode{},
		&wasmtypes.MsgStoreAndInstantiateContract{},
		&wasmtypes.MsgStoreAndMigrateContract{},
	} {
		if err := rejectWasmMsgs([]sdk.Msg{msg}, 0, true); err == nil {
			t.Errorf("%s was admitted with the feature on", sdk.MsgTypeURL(msg))
		}
		exec := authz.NewMsgExec(grantee, []sdk.Msg{&wasmtypes.MsgInstantiateContract{}, msg})
		if err := rejectWasmMsgs([]sdk.Msg{&exec}, 0, true); err == nil {
			t.Errorf("authz-wrapped %s next to an instantiation was admitted", sdk.MsgTypeURL(msg))
		}
	}
}

// The decorator reads the switch in CheckTx only: off it refuses instantiation
// as before, on it admits it, and block execution is never touched.
func TestWasmMempoolGuardDecorator_FollowsTheSwitch(t *testing.T) {
	tx := &memTx{msgs: []sdk.Msg{&wasmtypes.MsgInstantiateContract{}}}
	upload := &memTx{msgs: []sdk.Msg{&wasmtypes.MsgStoreCode{}}}
	checkTx := sdk.Context{}.WithIsCheckTx(true)
	deliverTx := sdk.Context{}

	for _, c := range []struct {
		name   string
		guard  WasmMempoolGuardDecorator
		ctx    sdk.Context
		tx     sdk.Tx
		admits bool
	}{
		{"nil switch, CheckTx", NewWasmMempoolGuardDecorator(nil), checkTx, tx, false},
		{"off, CheckTx", NewWasmMempoolGuardDecorator(featureSwitch(false)), checkTx, tx, false},
		{"on, CheckTx", NewWasmMempoolGuardDecorator(featureSwitch(true)), checkTx, tx, true},
		{"on, CheckTx, upload", NewWasmMempoolGuardDecorator(featureSwitch(true)), checkTx, upload, false},
		{"off, DeliverTx", NewWasmMempoolGuardDecorator(featureSwitch(false)), deliverTx, tx, true},
		{"off, DeliverTx, upload", NewWasmMempoolGuardDecorator(featureSwitch(false)), deliverTx, upload, true},
	} {
		n := 0
		_, err := c.guard.AnteHandle(c.ctx, c.tx, false, reached(&n))
		if c.admits && (err != nil || n != 1) {
			t.Errorf("%s: err %v, next reached %d times; want admitted", c.name, err, n)
		}
		if !c.admits && (err == nil || n != 0) {
			t.Errorf("%s: err %v, next reached %d times; want refused", c.name, err, n)
		}
	}
}

// The switch is the v3.2.5 feature, and the release turns it on.
func TestWasmGovernedDeployIsAV325Feature(t *testing.T) {
	k := &fakeFeatureKeeper{}
	sw := wasmGovernedDeployFeature(k)
	if sw(sdk.Context{}) {
		t.Fatal("switch on with no feature enabled")
	}
	k.enabled = map[string]bool{pqctypes.FeatureWasmGovernedDeploy: true}
	if !sw(sdk.Context{}) {
		t.Fatal("switch does not read FeatureWasmGovernedDeploy")
	}
	if wasmGovernedDeployFeature(nil) != nil {
		t.Fatal("a nil keeper must give a nil (off) switch")
	}
	for _, f := range pqctypes.ReleaseV3_2_5ProtocolFeatures {
		if f == pqctypes.FeatureWasmGovernedDeploy {
			return
		}
	}
	t.Fatal("FeatureWasmGovernedDeploy is not enabled by the v3.2.5 release")
}

// x/group executes a proposal's messages with the group policy as signer, so
// an upload inside a group proposal (submitted with Exec TRY) is an upload by
// an ordinary account and must not pass the guard.
func TestWasmMempoolGuard_BlocksGroupProposalUpload(t *testing.T) {
	policy := sdk.AccAddress("group_policy________").String()
	proposer := sdk.AccAddress("proposer____________").String()
	prop, err := group.NewMsgSubmitProposal(policy, []string{proposer},
		[]sdk.Msg{&wasmtypes.MsgStoreCode{}}, "", group.Exec_EXEC_TRY, "t", "s")
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectBlockedWasmMsgs([]sdk.Msg{prop}, 0); err == nil {
		t.Fatal("group-proposal MsgStoreCode was admitted")
	}
	// Also when the proposal itself sits inside an authz grant.
	exec := authz.NewMsgExec(sdk.AccAddress("grantee_____________"), []sdk.Msg{prop})
	if err := rejectBlockedWasmMsgs([]sdk.Msg{&exec}, 0); err == nil {
		t.Fatal("authz-wrapped group-proposal MsgStoreCode was admitted")
	}
	// And a group proposal inside a group proposal.
	outer, err := group.NewMsgSubmitProposal(policy, []string{proposer}, []sdk.Msg{prop}, "", group.Exec_EXEC_UNSPECIFIED, "t", "s")
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectBlockedWasmMsgs([]sdk.Msg{outer}, 0); err == nil {
		t.Fatal("nested group-proposal MsgStoreCode was admitted")
	}
	// From governed deploy an instantiate in a group proposal is admitted like
	// a top-level one; the upload stays refused.
	inst, err := group.NewMsgSubmitProposal(policy, []string{proposer},
		[]sdk.Msg{&wasmtypes.MsgInstantiateContract{}}, "", group.Exec_EXEC_TRY, "t", "s")
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectWasmMsgs([]sdk.Msg{inst}, 0, true); err != nil {
		t.Fatalf("governed deploy refused a group-proposal instantiate: %v", err)
	}
	if err := rejectWasmMsgs([]sdk.Msg{prop}, 0, true); err == nil {
		t.Fatal("governed deploy admitted a group-proposal upload")
	}
}

// A group proposal of ordinary messages passes.
func TestWasmMempoolGuard_AllowsBenignGroupProposal(t *testing.T) {
	policy := sdk.AccAddress("group_policy________").String()
	prop, err := group.NewMsgSubmitProposal(policy, []string{policy},
		[]sdk.Msg{&banktypes.MsgSend{}}, "", group.Exec_EXEC_TRY, "t", "s")
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectBlockedWasmMsgs([]sdk.Msg{prop}, 0); err != nil {
		t.Fatalf("benign group proposal was rejected: %v", err)
	}
}
