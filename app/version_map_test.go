package app

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// The live chains' version map lists only the dependency-injected modules. The
// repair must add the missing ones at their current version and must not touch
// the ones already recorded, which reflect the version their state was written
// at.
func TestMergeMissingModuleVersionsAddsOnlyWhatIsMissing(t *testing.T) {
	stored := module.VersionMap{"auth": 5, "bank": 4, "staking": 5}
	current := module.VersionMap{"auth": 6, "bank": 4, "staking": 5, "feemarket": 1, "evm": 3, "pqc": 1}

	got := mergeMissingModuleVersions(stored, current)

	if got["auth"] != 5 {
		t.Fatalf("auth was rewritten to %d; a recorded version must stay as stored", got["auth"])
	}
	for _, name := range []string{"feemarket", "evm", "pqc"} {
		if got[name] != current[name] {
			t.Fatalf("%s missing or wrong: got %d want %d", name, got[name], current[name])
		}
	}
	if len(got) != len(current) {
		t.Fatalf("merged map has %d entries, want %d", len(got), len(current))
	}
	if stored["feemarket"] != 0 {
		t.Fatal("the input map was mutated")
	}
}
