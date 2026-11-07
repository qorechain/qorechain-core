package types

import (
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Verified outer signers.
//
// Some messages are a statement made with the account's post-quantum key:
// opening or closing the EVM window, giving or clearing EIP-7702 consent, and
// managing an abstract account's authenticators and spending rules. The
// message names the account (Sender, Owner) and the Cosmos ante handler
// checks that this account signed the transaction, but only when the message
// is the transaction's own. A message dispatched from inside another one (an
// authz MsgExec, an x/group proposal, a contract's sub-message, an
// interchain-account packet) reaches its handler with no check of the key of
// the account it names: the grantee, the group members or the contract
// authorised it, not that account.
//
// The hybrid signature decorator therefore records, on the context it passes
// on, every outer signer whose post-quantum signature it verified (a
// registered key, or a PQC multisig member set that met its threshold). The
// context returned by the ante handler is the one the transaction's messages
// execute in, nested dispatch included, so a handler can ask whether the
// account its message names is one of them. Only a message the account
// itself signed with its post-quantum key passes: for a nested message the
// recorded signers are the outer transaction's, never the account the inner
// message names on their behalf (a group policy or a contract address cannot
// sign a transaction at all, and an authz granter did not sign this one).
//
// In simulation the decorator skips the cryptographic verification, as it
// skips it for the transaction itself, and records the signers whose
// verification delivery would run, so a simulated transaction executes as
// the delivered one would with a valid signature.
//
// Recording writes no state, costs no gas and emits nothing; the refusal is
// the handler's, behind its protocol feature.

type verifiedOuterSignersKey struct{}

// WithVerifiedOuterSigner returns ctx with addr added to the verified outer
// signers. The set already on ctx is copied, never modified in place, so a
// context derived earlier keeps the set it had.
func WithVerifiedOuterSigner(ctx sdk.Context, addr sdk.AccAddress) sdk.Context {
	if len(addr) == 0 || IsVerifiedOuterSigner(ctx, addr) {
		return ctx
	}
	prev := verifiedOuterSigners(ctx)
	next := make(map[string]struct{}, len(prev)+1)
	for k := range prev {
		next[k] = struct{}{}
	}
	next[string(addr)] = struct{}{}
	return ctx.WithValue(verifiedOuterSignersKey{}, next)
}

func verifiedOuterSigners(ctx sdk.Context) map[string]struct{} {
	if ctx.Context() == nil {
		return nil
	}
	m, _ := ctx.Value(verifiedOuterSignersKey{}).(map[string]struct{})
	return m
}

// IsVerifiedOuterSigner reports whether addr signed the transaction ctx
// belongs to and the ante handler verified its post-quantum signature.
func IsVerifiedOuterSigner(ctx sdk.Context, addr sdk.AccAddress) bool {
	if len(addr) == 0 {
		return false
	}
	_, ok := verifiedOuterSigners(ctx)[string(addr)]
	return ok
}

// RequireVerifiedOuterSigner refuses unless account (a bech32 address) is a
// verified outer signer of the transaction ctx belongs to. It is the check a
// handler of a post-quantum consent or account-management message runs, from
// FeaturePQCConsentOuterSigner, on the account the message names. It reads
// nothing from the store and charges no gas.
func RequireVerifiedOuterSigner(ctx sdk.Context, account string) error {
	addr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return ErrSignerNotPQCVerified.Wrapf("invalid account %q: %s", account, err)
	}
	if !IsVerifiedOuterSigner(ctx, addr) {
		return ErrSignerNotPQCVerified.Wrapf(
			"%s did not sign this transaction with its post-quantum key; this message must be the account's own "+
				"transaction message, not dispatched through authz, a group proposal, a contract or another module",
			account)
	}
	return nil
}

// ErrSignerNotPQCVerified is returned by RequireVerifiedOuterSigner.
var ErrSignerNotPQCVerified = errors.Register(ModuleName, 340,
	"the account this message names did not sign the transaction with its post-quantum key")
