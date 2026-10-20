//go:build !full

package pqc

import (
	"strings"
	"testing"
)

// QSR-2026-0356: the community build registers no pqc message types, so its
// pqc tx commands refuse instead of broadcasting what its node cannot decode.
func TestStubTxCommandsRefuse(t *testing.T) {
	root := AppModuleBasic{}.GetTxCmd()
	for _, sub := range []string{"register-key-v2", "register-key"} {
		cmd, _, err := root.Find([]string{sub})
		if err != nil {
			t.Fatal(err)
		}
		for c := cmd; c != nil; c = c.Parent() {
			if c.PersistentPreRunE != nil {
				err = c.PersistentPreRunE(cmd, nil)
				break
			}
		}
		if err == nil || !strings.Contains(err.Error(), "community build") {
			t.Fatalf("%s: got %v, want a community-build refusal", sub, err)
		}
	}
}
