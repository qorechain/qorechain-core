package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuthenticatorResolver resolves a foreign-scheme wallet key (e.g. a Phantom
// ed25519 key) to the canonical native account it is authorized to act for, and
// verifies its signatures. It is implemented by x/abstractaccount and wired into
// the SVM keeper via SetAuthenticatorResolver, so x/svm needs no import of that
// module. This is what lets a Phantom signer drive the user's single unified
// account (same identity + same balance) instead of a separate one.
type AuthenticatorResolver interface {
	// ResolveAuthenticatorAddr returns the 20-byte canonical account and its
	// granted permissions for an ACTIVE (scheme, pubkey) authenticator, or
	// ok=false if unbound / revoked / expired.
	ResolveAuthenticatorAddr(ctx sdk.Context, scheme string, pubkey []byte) (account []byte, permissions []string, ok bool)

	// VerifyForeignSignature verifies sig over msg for (scheme, pubkey). The
	// caller is responsible for domain separation (msg must bind chain-id +
	// account + nonce to prevent cross-chain / cross-account replay).
	VerifyForeignSignature(scheme string, pubkey, msg, sig []byte) bool
}
