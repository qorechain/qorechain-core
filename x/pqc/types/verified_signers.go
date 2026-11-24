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

// Top-level messages (FeaturePQCConsentOwnMessage).
//
// "The account signed this transaction" is weaker than "the account sent this
// message": a contract holding an authz grant from the account can dispatch
// a message in the account's name during a transaction the account signed
// (to call that contract), and the account is then a verified outer signer
// although it never signed that message. The hybrid signature decorator
// therefore also records the transaction's own top-level messages, as the
// very values the message router will hand to their handlers, and a handler
// can require that its message is one of them. A message built by authz, a
// group, a contract or another module is a different value and never
// matches. Recording writes no state and costs no gas.

type topLevelMessagesKey struct{}

// WithTopLevelMessages returns ctx carrying msgs as the transaction's own
// top-level messages.
func WithTopLevelMessages(ctx sdk.Context, msgs []sdk.Msg) sdk.Context {
	return ctx.WithValue(topLevelMessagesKey{}, append([]sdk.Msg(nil), msgs...))
}

// IsTopLevelMessage reports whether msg is one of the top-level messages of
// the transaction ctx belongs to: the same value, not an equal one.
func IsTopLevelMessage(ctx sdk.Context, msg sdk.Msg) bool {
	if msg == nil || ctx.Context() == nil {
		return false
	}
	msgs, _ := ctx.Value(topLevelMessagesKey{}).([]sdk.Msg)
	for _, m := range msgs {
		if m == msg {
			return true
		}
	}
	return false
}

// RequireOwnMessage refuses unless msg is one of the transaction's own
// top-level messages and account (a bech32 address, the account msg names)
// is a verified outer signer of the transaction. Since a top-level message is
// signed by every account it names as a signer, this is "the account signed
// this message with its post-quantum key". It reads nothing from the store
// and charges no gas.
func RequireOwnMessage(ctx sdk.Context, account string, msg sdk.Msg) error {
	if err := RequireVerifiedOuterSigner(ctx, account); err != nil {
		return err
	}
	if !IsTopLevelMessage(ctx, msg) {
		return ErrSignerNotPQCVerified.Wrapf(
			"%s did not send this message itself: it must be one of the account's own transaction messages, "+
				"not dispatched through authz, a group proposal, a contract or another module", account)
	}
	return nil
}
