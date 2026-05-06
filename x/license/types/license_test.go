package types

import (
	"testing"
)

// TestLicense_IsExpired covers the boundary conditions of the expiry check.
// Regression guard for the v2.17.0 audit fix that added the 100-block
// EndBlocker interval — the IsExpired predicate must remain monotonic.
func TestLicense_IsExpired(t *testing.T) {
	cases := []struct {
		name      string
		expiresAt int64
		height    int64
		want      bool
	}{
		{"never_expires", 0, 1_000_000, false},
		{"never_expires_at_zero", 0, 0, false},
		{"future_expiry", 200, 100, false},
		{"exact_expiry_block", 100, 100, true},
		{"past_expiry", 50, 100, true},
		{"expiry_one_below_height", 99, 100, true},
		{"expiry_one_above_height", 101, 100, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := License{ExpiresAt: c.expiresAt}
			if got := l.IsExpired(c.height); got != c.want {
				t.Errorf("IsExpired(expires=%d, height=%d) = %v, want %v",
					c.expiresAt, c.height, got, c.want)
			}
		})
	}
}

// TestLicense_IsActive covers the interaction of Suspended + ExpiresAt.
// An active license requires both: not suspended AND not expired.
func TestLicense_IsActive(t *testing.T) {
	cases := []struct {
		name      string
		suspended bool
		expiresAt int64
		height    int64
		want      bool
	}{
		{"clean_unbounded", false, 0, 100, true},
		{"clean_future_expiry", false, 200, 100, true},
		{"suspended_unbounded", true, 0, 100, false},
		{"suspended_future_expiry", true, 200, 100, false},
		{"expired_not_suspended", false, 50, 100, false},
		{"expired_and_suspended", true, 50, 100, false},
		{"exact_expiry_boundary", false, 100, 100, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := License{Suspended: c.suspended, ExpiresAt: c.expiresAt}
			if got := l.IsActive(c.height); got != c.want {
				t.Errorf("IsActive(suspended=%v, expires=%d, height=%d) = %v, want %v",
					c.suspended, c.expiresAt, c.height, got, c.want)
			}
		})
	}
}

// TestLicense_RoundTrip ensures Marshal/Unmarshal preserve all fields.
func TestLicense_RoundTrip(t *testing.T) {
	original := License{
		Grantee:   "qor1validator",
		FeatureID: FeatureBridgeEthereum,
		ExpiresAt: 1_234_567,
		GrantedAt: 1_000_000,
		GrantedBy: "qor1gov",
		Suspended: false,
		Metadata:  `{"reason":"audit-grant"}`,
	}
	bz, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	round, err := Unmarshal(bz)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if round != original {
		t.Errorf("round-trip mismatch:\noriginal: %+v\nround:    %+v", original, round)
	}
}

// TestLicense_SuspensionPreservedAcrossMarshal — defense-in-depth for the
// suspend/resume operations added in the v2.17.0 audit.
func TestLicense_SuspensionPreservedAcrossMarshal(t *testing.T) {
	l := License{Grantee: "qor1x", FeatureID: "bridge_ethereum", Suspended: true}
	bz, err := l.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	round, err := Unmarshal(bz)
	if err != nil {
		t.Fatal(err)
	}
	if !round.Suspended {
		t.Error("Suspended flag lost across marshal round-trip")
	}
	if round.IsActive(0) {
		t.Error("suspended license must not be active even at height 0")
	}
}
