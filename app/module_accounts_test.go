package app

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// Every module that mints or burns must have an account with that permission,
// or the first call fails at runtime with "module account does not exist" and
// whatever it was moving is stranded. The bridge module had no entry at all
// until v3.2.0, which would have broken the first deposit on the first
// activated route (QSR-2026-0225).
func TestMintingModulesHaveAccounts(t *testing.T) {
	perms := map[string][]string{}
	for _, a := range moduleAccPerms {
		perms[a.Account] = a.Permissions
	}
	for _, name := range []string{"bridge", "rdk", "amm", "erc20", "evm", "transfer", "precisebank", "xqore"} {
		got, ok := perms[name]
		if !ok {
			t.Fatalf("module %q has no account in the permission table", name)
		}
		var minter, burner bool
		for _, p := range got {
			minter = minter || p == authtypes.Minter
			burner = burner || p == authtypes.Burner
		}
		if !minter || !burner {
			t.Fatalf("module %q needs Minter and Burner, has %v", name, got)
		}
	}
	// Whatever is in the table is blocked from ordinary sends, except svm.
	blocked := map[string]bool{}
	for _, a := range blockAccAddrs {
		blocked[a] = true
	}
	if !blocked["bridge"] {
		t.Fatal("the bridge module account is not blocked from plain sends")
	}
	if blocked["svm"] {
		t.Fatal("svm must stay sendable")
	}
}
