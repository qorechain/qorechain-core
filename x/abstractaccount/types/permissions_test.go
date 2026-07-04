package types

import "testing"

func TestRequiredPermission_knownAndUnknown(t *testing.T) {
	cases := map[string]string{
		"/cosmos.bank.v1beta1.MsgSend":              PermSend,
		"/cosmos.staking.v1beta1.MsgDelegate":       PermDelegate,
		"/cosmos.gov.v1.MsgVote":                    PermVote,
		"/cosmos.evm.vm.v1.MsgEthereumTx":           PermEVM,
		"/qorechain.svm.v1.MsgExecuteProgram":       PermSVM,
		"/qorechain.amm.v1.MsgSwapExactIn":          PermAMM,
		"/ibc.applications.transfer.v1.MsgTransfer": PermIBC,
		"/qorechain.svm.v1.MsgDeployProgram":        PermDeploy,
	}
	for url, want := range cases {
		got, known := RequiredPermission(url)
		if !known || got != want {
			t.Fatalf("%s → (%q,%v), want (%q,true)", url, got, known, want)
		}
	}
	if _, known := RequiredPermission("/some.unknown.v1.MsgWhatever"); known {
		t.Fatal("unknown type URL must be unknown (fail-closed)")
	}
}

func TestMessageAllowed_failClosed(t *testing.T) {
	// scoped to send only
	send := []string{PermSend}
	if !MessageAllowed(send, "/cosmos.bank.v1beta1.MsgSend") {
		t.Fatal("send perm must allow MsgSend")
	}
	if MessageAllowed(send, "/cosmos.staking.v1beta1.MsgDelegate") {
		t.Fatal("send perm must NOT allow delegate")
	}
	if MessageAllowed(send, "/some.unknown.v1.MsgX") {
		t.Fatal("unknown msg must be denied (fail-closed)")
	}
	// all grants everything EXCEPT key-mgmt
	all := []string{PermAll}
	if !MessageAllowed(all, "/qorechain.svm.v1.MsgExecuteProgram") {
		t.Fatal("all must allow svm execute")
	}
	if MessageAllowed(all, "/qorechain.abstractaccount.v1.MsgRegisterAuthenticator") {
		t.Fatal("all must NOT allow key-management (register authenticator)")
	}
	if MessageAllowed(all, "/qorechain.pqc.v1.MsgRegisterPQCKeyV2") {
		t.Fatal("all must NOT allow key-management (register pqc key v2)")
	}
	// empty perms deny everything
	if MessageAllowed(nil, "/cosmos.bank.v1beta1.MsgSend") {
		t.Fatal("empty perms must deny")
	}
}

func TestKeyManagement_neverDelegable(t *testing.T) {
	for _, url := range []string{
		"/qorechain.abstractaccount.v1.MsgRegisterAuthenticator",
		"/qorechain.abstractaccount.v1.MsgRevokeAuthenticator",
		"/qorechain.pqc.v1.MsgRegisterPQCKeyV2",
		"/qorechain.pqc.v1.MsgRegisterPQCKey",
		"/qorechain.pqc.v1.MsgMigratePQCKey",
	} {
		if !IsKeyManagementMsg(url) {
			t.Fatalf("%s must be key-management (non-delegable)", url)
		}
	}
	if IsKeyManagementMsg("/cosmos.bank.v1beta1.MsgSend") {
		t.Fatal("MsgSend is not key-management")
	}
}

func TestValidPermissions(t *testing.T) {
	for _, p := range AllPermissions() {
		if !IsValidPermission(p) {
			t.Fatalf("%s should be valid", p)
		}
	}
	if IsValidPermission("bogus") {
		t.Fatal("bogus is not a valid permission")
	}
}
