package types

import "testing"

// Every protocol feature name the binary knows appears exactly once across the
// released, staged and chain-scoped lists. Release branches each append their
// own list to a release; a name repeated across two of them (or a list
// appended twice in a merge) would be enabled twice by a handler and would
// make the lists disagree about which release a feature belongs to.
func TestProtocolFeatureNamesUnique(t *testing.T) {
	seen := map[string]string{}
	check := func(list string, names []string) {
		for _, f := range names {
			if f == "" {
				t.Fatalf("empty feature name in %s", list)
			}
			if prev, dup := seen[f]; dup {
				t.Fatalf("feature %q listed in %s and again in %s", f, prev, list)
			}
			seen[f] = list
		}
	}
	check("AllProtocolFeatures", AllProtocolFeatures)
	check("StagedProtocolFeatures", StagedProtocolFeatures)
	check("ChainScopedProtocolFeatures", ChainScopedProtocolFeatures)
}

// Each v3.2.8 area list is part of the v3.2.8 release exactly once, and the
// chain-scoped features are not released on every chain.
func TestV328ReleaseListsComposeOnce(t *testing.T) {
	inRelease := map[string]int{}
	for _, f := range ReleaseV3_2_8ProtocolFeatures {
		inRelease[f]++
	}
	areas := map[string][]string{
		"PQCAuthz":   V328PQCAuthzFeatures,
		"AA":         V328AAFeatures,
		"Econ":       V328EconFeatures,
		"RDK":        V328RDKFeatures,
		"RDKData":    V328RDKDataFeatures,
		"RDKModes":   V328RDKModeFeatures,
		"AI":         V328AIFeatures,
		"SVM":        V328SVMFeatures,
		"QSR2":       V328QSR2Features,
		"WasmFilter": V328WasmFilterFeatures,
		"AMMRLAI":    V328AMMRLAIFeatures,
		"WasmAuthz":  V328WasmAuthzFeatures,
		"EVMAuthz":   V328EVMAuthzFeatures,
		"SVMAuthz":   V328SVMAuthzFeatures,
		"XVMAuthz":   V328XVMAuthzFeatures,
	}
	total := 0
	for area, l := range areas {
		if len(l) == 0 {
			t.Fatalf("v3.2.8 area %s has no features", area)
		}
		for _, f := range l {
			if inRelease[f] != 1 {
				t.Fatalf("%s feature %q is in the v3.2.8 release %d times", area, f, inRelease[f])
			}
		}
		total += len(l)
	}
	if total != len(ReleaseV3_2_8ProtocolFeatures) {
		t.Fatalf("v3.2.8 release has %d features, the area lists hold %d", len(ReleaseV3_2_8ProtocolFeatures), total)
	}
	for _, f := range ChainScopedProtocolFeatures {
		if inRelease[f] != 0 {
			t.Fatalf("chain-scoped feature %q is in the v3.2.8 release", f)
		}
		if !KnownProtocolFeature(f) {
			t.Fatalf("chain-scoped feature %q is not known", f)
		}
	}
}
