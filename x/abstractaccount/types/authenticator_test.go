package types

import (
	"testing"
	"time"
)

func TestValidateAuthenticator(t *testing.T) {
	valid := Authenticator{
		Scheme:      SchemeEd25519,
		PubKey:      make([]byte, 32),
		Permissions: []string{"send", "svm"},
		Expiry:      time.Now().Add(24 * time.Hour),
	}
	if err := ValidateAuthenticator(valid); err != nil {
		t.Fatalf("expected valid ed25519 authenticator, got %v", err)
	}

	cases := []struct {
		name string
		a    Authenticator
	}{
		{"bad scheme", Authenticator{Scheme: "rsa", PubKey: make([]byte, 32), Permissions: []string{"send"}, Expiry: time.Now().Add(time.Hour)}},
		{"short ed25519 key", Authenticator{Scheme: SchemeEd25519, PubKey: make([]byte, 31), Permissions: []string{"send"}, Expiry: time.Now().Add(time.Hour)}},
		{"no perms", Authenticator{Scheme: SchemeEd25519, PubKey: make([]byte, 32), Expiry: time.Now().Add(time.Hour)}},
		{"zero expiry", Authenticator{Scheme: SchemeEd25519, PubKey: make([]byte, 32), Permissions: []string{"send"}}},
	}
	for _, c := range cases {
		if err := ValidateAuthenticator(c.a); err == nil {
			t.Errorf("%s: expected validation error, got nil", c.name)
		}
	}
}

// "ttl too long" used to be one of the cases above. It was removed deliberately,
// not because the rule was dropped but because it moved.
//
// ValidateAuthenticator is reached from ValidateBasic, which runs in the ante chain
// during block finalisation. Judging a TTL there meant comparing against the node's
// own wall clock, so a registration tuned to sit on the boundary was accepted by
// validators whose clock put it under the limit and rejected by those whose clock put
// it over: divergent block results and a halted chain, for the price of one
// transaction. The bound is now enforced in the handler against block time, which is
// the same number on every validator executing the same block.
//
// This test pins both halves, so neither can quietly disappear: ValidateBasic must
// NOT judge the TTL, and the block-time check must.
func TestTTLIsJudgedAgainstBlockTimeAndNotInValidateBasic(t *testing.T) {
	tooLong := Authenticator{
		Scheme:      SchemeEd25519,
		PubKey:      make([]byte, 32),
		Permissions: []string{"send"},
		Expiry:      time.Now().Add(60 * 24 * time.Hour),
	}

	if err := ValidateAuthenticator(tooLong); err != nil {
		t.Fatalf("ValidateBasic judges the TTL again, which is the consensus hazard "+
			"this was moved to avoid: %v", err)
	}

	blockTime := time.Now()
	if err := ValidateAuthenticatorTTL(tooLong.Expiry, blockTime); err == nil {
		t.Fatal("the TTL bound is enforced nowhere at all now")
	}
}

func TestAuthenticatorActiveAndPerms(t *testing.T) {
	now := time.Now()
	a := Authenticator{Scheme: SchemeEd25519, Expiry: now.Add(time.Hour), Permissions: []string{"send", "svm"}}
	if !a.IsActive(now) {
		t.Fatal("should be active before expiry")
	}
	if a.IsActive(now.Add(2 * time.Hour)) {
		t.Fatal("should be inactive after expiry")
	}
	a.Revoked = true
	if a.IsActive(now) {
		t.Fatal("revoked authenticator must be inactive")
	}
	if !a.HasPermission("svm") || a.HasPermission("delegate") {
		t.Fatal("permission check wrong")
	}
}
