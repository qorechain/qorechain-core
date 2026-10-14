package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	// The TTL bound is NOT checked here. See ValidateAuthenticatorTTL.
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
		Enabled:           true, // v3.1.85: authenticator execution active by default (QoreX §8.13 feature-probe)
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
		// Three accepted forms:
		//   33 / 65 — compressed / uncompressed SEC1 public key. Only the 33-byte
		//             form verifies (cosmos secp256k1 scheme, sha256-based); a
		//             65-byte key was accepted here but never verifies, and is
		//             refused at registration from the abstractaccount-hardening
		//             protocol feature on (ValidateAuthenticatorStrict).
		//   20      — an Ethereum address (keccak of the pubkey), verified with the
		//             EIP-191 personal_sign scheme (keccak + ecrecover). This is the
		//             form a browser wallet like MetaMask can produce, since it only
		//             exposes the address + personal_sign, not the raw pubkey.
		if len(a.PubKey) != 33 && len(a.PubKey) != 65 && len(a.PubKey) != 20 {
			return fmt.Errorf("secp256k1 pubkey must be 33, 65, or 20 (eth address) bytes, got %d", len(a.PubKey))
		}
	default:
		return fmt.Errorf("unsupported authenticator scheme %q", a.Scheme)
	}
	if a.Expiry.IsZero() {
		return fmt.Errorf("authenticator expiry cannot be zero")
	}
	// The TTL bound is NOT checked here. See ValidateAuthenticatorTTL.
	if len(a.Permissions) == 0 {
		return fmt.Errorf("authenticator must have at least one permission")
	}
	return nil
}

// ValidateAuthenticatorStrict is ValidateAuthenticator plus the checks the
// keeper applies at registration from the abstractaccount-hardening protocol
// feature on. ValidateBasic cannot tell the height, so it keeps calling
// ValidateAuthenticator alone.
//   - A 65-byte (uncompressed) secp256k1 key is refused: verification handles
//     only the 33-byte and 20-byte forms, so such a key could be registered
//     and paid for but never sign (QSR-2026-0403). Register it compressed.
//   - Every permission must be a known one (IsValidPermission). An unknown
//     string granted nothing, but it was stored and shown as if it did
//     (QSR-2026-0436 a).
func ValidateAuthenticatorStrict(a Authenticator) error {
	if err := ValidateAuthenticator(a); err != nil {
		return err
	}
	if a.Scheme == SchemeSecp256k1 && len(a.PubKey) == 65 {
		return fmt.Errorf("uncompressed (65-byte) secp256k1 keys cannot verify; register the 33-byte compressed key")
	}
	for _, p := range a.Permissions {
		if !IsValidPermission(p) {
			return fmt.Errorf("unknown permission %q", p)
		}
	}
	return nil
}

// ValidateAccountAddress checks an AccountAddress field is a bech32 address
// (QSR-2026-0436 b). The messages' ValidateBasic only checks that it is not
// empty; the handler applies this from the abstractaccount-hardening protocol
// feature on.
func ValidateAccountAddress(addr string) error {
	if _, err := sdk.AccAddressFromBech32(addr); err != nil {
		return fmt.Errorf("invalid account address: %w", err)
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

// ValidateAuthenticatorTTL bounds how far in the future an expiry may sit, measured
// against BLOCK TIME.
//
// WHY THIS IS NOT IN ValidateBasic, WHICH IS WHERE IT USED TO BE. ValidateBasic runs
// inside the ante chain, which executes during block finalisation, where every
// validator must reach the same answer for the same transaction. The check used
// time.Until, i.e. the node's own wall clock. A registration whose expiry sits
// within milliseconds of the boundary is then accepted by validators whose clock puts
// it under the limit and rejected by those whose clock puts it over. Different
// validators produce different block results, the app hashes diverge, and the chain
// halts. Submitting one message with a tuned expiry is enough to trigger it, from
// anywhere, for the price of one transaction.
//
// Block time is the same number for every validator executing the same block, which
// is the whole reason consensus code reads it and never the wall clock. The check
// therefore moves to the handler, where a context exists.
//
// The same rule already had to be learned once in this codebase: x/ai carries a
// comment about using block time "instead of time.Now() which breaks consensus".
// This is that rule in the one place it had not been applied.
func ValidateAuthenticatorTTL(expiry, blockTime time.Time) error {
	if expiry.IsZero() {
		return fmt.Errorf("expiry cannot be zero")
	}
	if ttl := expiry.Sub(blockTime); ttl > MaxSessionTTL {
		return fmt.Errorf("TTL %v exceeds maximum %v", ttl, MaxSessionTTL)
	}
	return nil
}
