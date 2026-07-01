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
		{"ttl too long", Authenticator{Scheme: SchemeEd25519, PubKey: make([]byte, 32), Permissions: []string{"send"}, Expiry: time.Now().Add(60 * 24 * time.Hour)}},
	}
	for _, c := range cases {
		if err := ValidateAuthenticator(c.a); err == nil {
			t.Errorf("%s: expected validation error, got nil", c.name)
		}
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
