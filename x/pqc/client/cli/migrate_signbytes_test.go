package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// QSR-2026-0129 (migrate part, fixed in v3.2.0 by 78045e9): the migrate-key
// help text told wallets to sign one preimage while the keeper verified
// another. Both now name types.MigrationSignBytes; this keeps them on the same
// function so they cannot drift apart again. The keeper lives in the private
// repository, so that half runs when QORE_HANDLER_ROOTS points at it (the full
// test target sets it) and is skipped otherwise.
func TestMigrateKeyHelpAndKeeperShareSignBytes(t *testing.T) {
	const fn = "types.MigrationSignBytes("
	cli, err := os.ReadFile("tx.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cli), fn[:len(fn)-1]) {
		t.Fatalf("migrate-key help no longer names %s", fn)
	}
	root := os.Getenv("QORE_HANDLER_ROOTS")
	if root == "" {
		t.Skip("QORE_HANDLER_ROOTS not set: keeper half not checked")
	}
	for _, r := range filepath.SplitList(root) {
		bz, err := os.ReadFile(filepath.Join(r, "x", "pqc", "keeper", "msg_server.go"))
		if err != nil {
			continue
		}
		if !strings.Contains(string(bz), fn) {
			t.Fatalf("the keeper's migrate handler no longer calls %s", fn)
		}
		return
	}
	t.Fatalf("x/pqc/keeper/msg_server.go not found under QORE_HANDLER_ROOTS=%s", root)
}
