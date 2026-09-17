package types

import (
	"encoding/json"
	"testing"
)

// A genesis in the shape both networks hold today lists no protocol feature,
// and must decode to none: InitGenesis then writes no feature record at
// height 0, which is what keeps a replay from the real genesis byte-identical
// to the history. A chain born on this binary lists the released features and
// none of the staged ones.
func TestProtocolFeaturesComeFromTheGenesisNotTheBinary(t *testing.T) {
	var gs GenesisState
	if err := json.Unmarshal([]byte(`{"params":{"min_security_level":5,"hybrid_signature_mode":0},"accounts":[],"stats":{},"algorithms":[]}`), &gs); err != nil {
		t.Fatal(err)
	}
	if len(gs.ProtocolFeatures) != 0 {
		t.Fatalf("a pre-upgrade genesis decoded with features %v", gs.ProtocolFeatures)
	}
	if err := gs.Validate(); err != nil {
		t.Fatalf("the pre-upgrade genesis was refused: %v", err)
	}

	def := DefaultGenesisState()
	if len(def.ProtocolFeatures) != len(AllProtocolFeatures) {
		t.Fatalf("default genesis lists %v, want the released set %v", def.ProtocolFeatures, AllProtocolFeatures)
	}
	for _, f := range def.ProtocolFeatures {
		if f == FeatureEVMPQCAuthorisationWindow {
			t.Fatal("a staged feature is enabled at genesis")
		}
	}

	bad := *def
	bad.ProtocolFeatures = append(bad.ProtocolFeatures, "no-such-feature")
	if err := bad.Validate(); err == nil {
		t.Fatal("an unknown feature name was accepted")
	}
	dup := *def
	dup.ProtocolFeatures = append(dup.ProtocolFeatures, AllProtocolFeatures[0])
	if err := dup.Validate(); err == nil {
		t.Fatal("a duplicated feature name was accepted")
	}
}
