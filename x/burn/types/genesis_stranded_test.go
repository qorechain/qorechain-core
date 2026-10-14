package types

import (
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/math"
)

// QSR-2026-0473: the stranded share record is part of the genesis state. It
// is optional (older files have no field), validated when present, and
// omitted from the JSON when absent.
func TestGenesisStrandedRewards(t *testing.T) {
	gs := DefaultGenesisState()
	if gs.StrandedRewards != nil {
		t.Fatal("default genesis carries a stranded record")
	}
	bz, err := json.Marshal(gs)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bz), "stranded_rewards") {
		t.Fatalf("absent record is serialised: %s", bz)
	}

	// A file written before the field existed still parses and validates.
	var old GenesisState
	if err := json.Unmarshal(bz, &old); err != nil || old.StrandedRewards != nil {
		t.Fatalf("old file: %v %+v", err, old.StrandedRewards)
	}

	gs.StrandedRewards = &StrandedRewards{
		Reserved: math.NewInt(6_693_045_921_330), Released: math.ZeroInt(),
		Initialized: true, SnapshotHeight: 2_981_742,
	}
	if err := gs.Validate(); err != nil {
		t.Fatal(err)
	}
	bz, err = json.Marshal(gs)
	if err != nil {
		t.Fatal(err)
	}
	var back GenesisState
	if err := json.Unmarshal(bz, &back); err != nil {
		t.Fatal(err)
	}
	if back.StrandedRewards == nil || !back.StrandedRewards.Initialized ||
		!back.StrandedRewards.Reserved.Equal(gs.StrandedRewards.Reserved) ||
		back.StrandedRewards.SnapshotHeight != 2_981_742 {
		t.Fatalf("round trip lost the record: %+v", back.StrandedRewards)
	}

	for name, bad := range map[string]StrandedRewards{
		"negative reserved": {Reserved: math.NewInt(-1), Released: math.ZeroInt(), Initialized: true},
		"negative released": {Reserved: math.ZeroInt(), Released: math.NewInt(-1), Initialized: true},
		"negative height":   {Reserved: math.ZeroInt(), Released: math.ZeroInt(), Initialized: true, SnapshotHeight: -1},
	} {
		g := DefaultGenesisState()
		b := bad
		g.StrandedRewards = &b
		if err := g.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
