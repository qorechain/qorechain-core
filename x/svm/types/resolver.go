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

	// AuthorizeAction (v3.1.84) is the single authorization gate: it verifies the
	// signature, resolves the ACTIVE authenticator, checks it is scoped for
	// requiredPerm via the canonical permission taxonomy (fail-closed), enforces
	// its SpendingRule against the base-unit outflow (uqor; caller converts lane
	// value), records the spend, and returns the 20-byte canonical account. err
	// is a typed chain error on any denial. outflow may be nil (permission-only).
	AuthorizeAction(ctx sdk.Context, scheme string, pubkey, msg, sig []byte, requiredPerm string, outflow sdk.Coins) (account []byte, err error)

	// EnforceAuthenticatorSpend (v3.1.85) charges a post-execution outflow against
	// the authenticator's SpendingRule and records it. It exists because an SVM
	// program's value movement is not statically known before execution: the SVM
	// lane authorizes permission scope up-front via AuthorizeAction(outflow=nil),
	// then measures the account's realized native-balance delta and calls this to
	// enforce the amount-limit. Returns a typed ErrSpendingLimitExceeded when the
	// outflow would breach the rule (the caller returns the error so the whole tx —
	// including the balance move — reverts). A nil/zero outflow is a no-op.
	EnforceAuthenticatorSpend(ctx sdk.Context, scheme string, pubkey []byte, outflow sdk.Coins) error
}
