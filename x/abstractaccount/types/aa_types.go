package types

import (
	"fmt"
	"time"
)

const MaxSessionTTL = 30 * 24 * time.Hour // 30 days maximum

// ValidateSessionKey checks that a session key is well-formed.
func ValidateSessionKey(key SessionKey) error {
	if key.Key == "" {
		return fmt.Errorf("session key cannot be empty")
	}
	if key.Expiry.IsZero() {
		return fmt.Errorf("session key expiry cannot be zero")
	}
	ttl := time.Until(key.Expiry)
	if ttl > MaxSessionTTL {
		return fmt.Errorf("session key TTL %v exceeds maximum %v", ttl, MaxSessionTTL)
	}
	if len(key.Permissions) == 0 {
		return fmt.Errorf("session key must have at least one permission")
	}
	return nil
}

// AbstractAccountConfig holds the module configuration.
type AbstractAccountConfig struct {
	Enabled           bool  `json:"enabled"`
	MaxSessionKeys    int   `json:"max_session_keys"`
	MaxSpendingRules  int   `json:"max_spending_rules"`
	DefaultSessionTTL int64 `json:"default_session_ttl"` // seconds
}

// DefaultAbstractAccountConfig returns default configuration.
func DefaultAbstractAccountConfig() AbstractAccountConfig {
	return AbstractAccountConfig{
		Enabled:           false,
		MaxSessionKeys:    10,
		MaxSpendingRules:  5,
		DefaultSessionTTL: 86400, // 24 hours
	}
}

// AbstractAccount represents a smart-contract backed account.
type AbstractAccount struct {
	Address         string          `json:"address"`
	ContractAddress string          `json:"contract_address"`
	AccountType     string          `json:"account_type"` // multisig, social_recovery, session_based
	SpendingRules   []SpendingRule  `json:"spending_rules"`
	SessionKeys     []SessionKey    `json:"session_keys"`
	Authenticators  []Authenticator `json:"authenticators"`
	CreatedAt       time.Time       `json:"created_at"`
	Owner           string          `json:"owner"`
}

// Authenticator scheme identifiers.
const (
	SchemeEd25519   = "ed25519"   // e.g. Phantom / any Solana wallet
	SchemeSecp256k1 = "secp256k1" // e.g. MetaMask / any EVM or Cosmos wallet
)

// Authenticator is a foreign-scheme public key that is authorized to act for a
// canonical QoreChain account, WITHOUT that key becoming its own identity. It
// lets a user drive their single account (one identity, one balance) from any
// wallet — including Phantom (ed25519) — under least-privilege, revocable,
// time-bounded terms.
//
// Security posture (see docs/plans/2026-06-30-svm-native-qor-unification.md):
//   - least privilege: Permissions + optional SpendingRule bound what it can do
//   - bounded exposure: Expiry (≤ MaxSessionTTL) auto-lapses the grant
//   - instant revocation: Revoked flips it off with no dependence on the key
//   - only the canonical account owner (root key) may register/revoke
type Authenticator struct {
	Scheme      string        `json:"scheme"`      // SchemeEd25519 | SchemeSecp256k1
	PubKey      []byte        `json:"pubkey"`      // raw public key bytes
	Permissions []string      `json:"permissions"` // send, delegate, vote, svm, ...
	SpendRule   *SpendingRule `json:"spend_rule"`  // optional per-key limit; nil = account default
	Expiry      time.Time     `json:"expiry"`      // hard lapse; must be ≤ now+MaxSessionTTL
	Label       string        `json:"label"`
	CreatedAt   time.Time     `json:"created_at"`
	Revoked     bool          `json:"revoked"`
}

// ValidateAuthenticator checks a foreign-scheme authenticator is well-formed and
// within policy (known scheme, correct key length, bounded TTL, non-empty perms).
func ValidateAuthenticator(a Authenticator) error {
	switch a.Scheme {
	case SchemeEd25519:
		if len(a.PubKey) != 32 {
			return fmt.Errorf("ed25519 pubkey must be 32 bytes, got %d", len(a.PubKey))
		}
	case SchemeSecp256k1:
		// compressed (33) or uncompressed (65) SEC1 encodings
		if len(a.PubKey) != 33 && len(a.PubKey) != 65 {
			return fmt.Errorf("secp256k1 pubkey must be 33 or 65 bytes, got %d", len(a.PubKey))
		}
	default:
		return fmt.Errorf("unsupported authenticator scheme %q", a.Scheme)
	}
	if a.Expiry.IsZero() {
		return fmt.Errorf("authenticator expiry cannot be zero")
	}
	if ttl := time.Until(a.Expiry); ttl > MaxSessionTTL {
		return fmt.Errorf("authenticator TTL %v exceeds maximum %v", ttl, MaxSessionTTL)
	}
	if len(a.Permissions) == 0 {
		return fmt.Errorf("authenticator must have at least one permission")
	}
	return nil
}

// IsActive reports whether the authenticator may currently be used.
func (a Authenticator) IsActive(now time.Time) bool {
	return !a.Revoked && !now.After(a.Expiry)
}

// HasPermission reports whether the authenticator is granted the given action.
func (a Authenticator) HasPermission(perm string) bool {
	for _, p := range a.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// SpendingRule is generated from proto/qorechain/abstractaccount/v1/tx.proto
// (see tx.pb.go). AbstractAccount.SpendingRules references the generated type.

// SessionKey represents a temporary key with limited permissions.
type SessionKey struct {
	Key         string    `json:"key"`
	Expiry      time.Time `json:"expiry"`
	Permissions []string  `json:"permissions"` // send, delegate, vote, etc.
	Label       string    `json:"label"`
	CreatedAt   time.Time `json:"created_at"`
}

// IsExpired checks if a session key has expired.
func (sk SessionKey) IsExpired(now time.Time) bool {
	return now.After(sk.Expiry)
}
