package types

import (
	"bytes"
	"context"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestVerifiedOuterSigners(t *testing.T) {
	a := sdk.AccAddress([]byte("signer-a____________"))
	b := sdk.AccAddress([]byte("signer-b____________"))

	// A context with no Go context at all (zero value) answers no.
	require.False(t, IsVerifiedOuterSigner(sdk.Context{}, a))

	base := sdk.NewContext(nil, cmtproto.Header{}, false, nil)
	require.False(t, IsVerifiedOuterSigner(base, a))
	require.ErrorIs(t, RequireVerifiedOuterSigner(base, a.String()), ErrSignerNotPQCVerified)

	withA := WithVerifiedOuterSigner(base, a)
	require.True(t, IsVerifiedOuterSigner(withA, a))
	require.False(t, IsVerifiedOuterSigner(withA, b))
	require.NoError(t, RequireVerifiedOuterSigner(withA, a.String()))
	require.ErrorIs(t, RequireVerifiedOuterSigner(withA, b.String()), ErrSignerNotPQCVerified)

	// Adding b copies the set: the context that only had a keeps only a.
	withAB := WithVerifiedOuterSigner(withA, b)
	require.True(t, IsVerifiedOuterSigner(withAB, a))
	require.True(t, IsVerifiedOuterSigner(withAB, b))
	require.False(t, IsVerifiedOuterSigner(withA, b))
	require.False(t, IsVerifiedOuterSigner(base, a))

	// Empty and malformed addresses are never verified.
	require.False(t, IsVerifiedOuterSigner(withAB, nil))
	require.Equal(t, withAB, WithVerifiedOuterSigner(withAB, nil))
	require.ErrorIs(t, RequireVerifiedOuterSigner(withAB, "not-bech32"), ErrSignerNotPQCVerified)
}

// RequireOwnMessage: the very value of one of the transaction's top-level
// messages, and the account a verified outer signer. An equal message built
// elsewhere (by authz, a group, a contract) is not the transaction's own.
func TestRequireOwnMessage(t *testing.T) {
	acct := sdk.AccAddress(bytes.Repeat([]byte{9}, 20))
	own := &MsgCloseEVMWindow{Sender: acct.String()}
	copyOf := *own
	base := sdk.Context{}.WithContext(context.Background())

	if err := RequireOwnMessage(base, acct.String(), own); err == nil {
		t.Fatal("nothing recorded: admitted")
	}
	signed := WithVerifiedOuterSigner(base, acct)
	if err := RequireOwnMessage(signed, acct.String(), own); err == nil {
		t.Fatal("signer verified, no top-level messages recorded: admitted")
	}
	both := WithTopLevelMessages(signed, []sdk.Msg{own})
	if err := RequireOwnMessage(both, acct.String(), own); err != nil {
		t.Fatalf("own top-level message: %v", err)
	}
	if err := RequireOwnMessage(both, acct.String(), &copyOf); err == nil {
		t.Fatal("an equal message that is not the transaction's own: admitted")
	}
	if err := RequireOwnMessage(WithTopLevelMessages(base, []sdk.Msg{own}), acct.String(), own); err == nil {
		t.Fatal("top-level message of an account that did not sign: admitted")
	}
}
