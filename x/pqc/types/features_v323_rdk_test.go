package types

import "testing"

func TestV323RdkFeatures(t *testing.T) {
	if len(V323RdkFeatures) != 1 || V323RdkFeatures[0] != FeatureRDKEscrowLedger {
		t.Fatalf("V323RdkFeatures = %v", V323RdkFeatures)
	}
	for _, list := range [][]string{ReleaseV3_2_0ProtocolFeatures, ReleaseV3_2_2ProtocolFeatures} {
		for _, f := range list {
			if f == FeatureRDKEscrowLedger {
				t.Fatal("the escrow-ledger switch must not be in an earlier release, or history would not replay")
			}
		}
	}
}
