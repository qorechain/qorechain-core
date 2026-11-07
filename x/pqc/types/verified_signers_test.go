package types

import (
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
