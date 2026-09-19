package types

import "testing"

// A staged feature must not be enabled by any path in this binary: not at
// genesis, not by the release handler. Both walk AllProtocolFeatures, so the
// two lists have to be disjoint, and the EVM-lane window has to be on the
// staged side until the release that switches it on.
// The EVM-lane post-quantum window is RELEASED in v3.2.0. Reopening the EVM
// lane at the upgrade height without it would restore a lane on which a
// classical key alone spends a post-quantum account's whole balance.
func TestEVMWindowIsReleased(t *testing.T) {
	released := map[string]bool{}
	for _, f := range AllProtocolFeatures {
		released[f] = true
	}
	if !released[FeatureEVMPQCAuthorisationWindow] {
		t.Fatal("the EVM-lane post-quantum window is not released; the upgrade would reopen the lane unprotected")
	}
	for _, f := range StagedProtocolFeatures {
		if released[f] {
			t.Fatalf("%q is both released and staged", f)
		}
	}
	if !KnownProtocolFeature(FeatureEVMPQCAuthorisationWindow) {
		t.Fatal("the feature is not known to the binary")
	}
}
