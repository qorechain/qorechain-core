package types

import "testing"

// A staged feature must not be enabled by any path in this binary: not at
// genesis, not by the release handler. Both walk AllProtocolFeatures, so the
// two lists have to be disjoint, and the EVM-lane window has to be on the
// staged side until the release that switches it on.
func TestEVMWindowIsStagedNotReleased(t *testing.T) {
	released := map[string]bool{}
	for _, f := range AllProtocolFeatures {
		released[f] = true
	}
	for _, f := range StagedProtocolFeatures {
		if released[f] {
			t.Fatalf("%q is both released and staged", f)
		}
	}
	if released[FeatureEVMPQCAuthorisationWindow] {
		t.Fatal("the EVM-lane post-quantum window is released; it is staged for the next release")
	}
	staged := false
	for _, f := range StagedProtocolFeatures {
		staged = staged || f == FeatureEVMPQCAuthorisationWindow
	}
	if !staged {
		t.Fatal("the EVM-lane window is in neither list, so nothing could ever switch it on")
	}
}
