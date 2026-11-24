//go:build full

package cli

import (
	"errors"
	"testing"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

func TestAutoSignBytes(t *testing.T) {
	noPlans := func(string) (int64, error) { return 0, nil }
	failPlans := func(string) (int64, error) { t.Fatal("applied_plan asked although the features answered"); return 0, nil }
	features := func(fs ...*types.ProtocolFeatureStatus) func() ([]*types.ProtocolFeatureStatus, error) {
		return func() ([]*types.ProtocolFeatureStatus, error) { return fs, nil }
	}
	v2on := &types.ProtocolFeatureStatus{Name: types.FeaturePQCSignBytesV2, Enabled: true, Known: true}
	v2off := &types.ProtocolFeatureStatus{Name: types.FeaturePQCSignBytesV2, Enabled: false, Known: true}
	other := &types.ProtocolFeatureStatus{Name: "some-other-switch", Enabled: true, Known: true}

	// A fresh chain under a public chain-id, every switch on from genesis and
	// no plan ever applied: the features say v2 (the plan oracle said v1).
	for _, id := range []string{"qorechain-diana", "qorechain-vladi", "devnet-1"} {
		if v, err := autoSignBytes(id, features(other, v2on), failPlans); err != nil || v != 2 {
			t.Fatalf("%s, switch on: v=%d err=%v", id, v, err)
		}
		if v, err := autoSignBytes(id, features(other, v2off), failPlans); err != nil || v != 1 {
			t.Fatalf("%s, switch listed off: v=%d err=%v", id, v, err)
		}
		if v, err := autoSignBytes(id, features(other), failPlans); err != nil || v != 1 {
			t.Fatalf("%s, switch absent: v=%d err=%v", id, v, err)
		}
	}

	// The query fails (an older node): the applied-plan check answers, asking
	// every name the switch shipped under.
	unavailable := func() ([]*types.ProtocolFeatureStatus, error) { return nil, errors.New("unknown service") }
	asked := map[string]bool{}
	if _, err := autoSignBytes("qorechain-diana", unavailable, func(n string) (int64, error) { asked[n] = true; return 0, nil }); err != nil {
		t.Fatal(err)
	}
	for _, n := range types.SignBytesV2Upgrades {
		if !asked[n] {
			t.Fatalf("fallback did not ask for %s", n)
		}
	}
	want := types.SignBytesVersionForApplied("qorechain-diana", map[string]int64{})
	if v, _ := autoSignBytes("qorechain-diana", unavailable, noPlans); v != want {
		t.Fatalf("fallback, nothing applied: v=%d want %d", v, want)
	}
	applied := func(n string) (int64, error) {
		if n == types.SignBytesV2Upgrades[0] {
			return 100, nil
		}
		return 0, nil
	}
	if v, _ := autoSignBytes("qorechain-diana", unavailable, applied); v != 2 {
		t.Fatalf("fallback, plan applied: v=%d", v)
	}
	if _, err := autoSignBytes("qorechain-diana", unavailable, func(string) (int64, error) { return 0, errors.New("down") }); err == nil {
		t.Fatal("both queries failing must be an error")
	}
}
