package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/crypto/ethsecp256k1"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
)

// QSR-2026-0491: the simulate path amino-encodes a StdSignature carrying the
// signer's stored pubkey with legacy.Cdc. Without the eth key types that
// failed for every eth_secp256k1 account that had signed before.
func TestRegisterEthKeysOnLegacyAmino(t *testing.T) {
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	sig := legacytx.StdSignature{PubKey: priv.PubKey(), Signature: make([]byte, 64)} //nolint:staticcheck

	// A codec without the registration fails the way the node did.
	unregistered := codec.NewLegacyAmino()
	cryptocodec.RegisterCrypto(unregistered)
	_, err = unregistered.Marshal(sig)
	require.ErrorContains(t, err, "unregistered concrete type")

	bare := codec.NewLegacyAmino()
	cryptocodec.RegisterCrypto(bare)
	RegisterEthKeysOnLegacyAmino(bare)
	_, err = bare.Marshal(sig)
	require.NoError(t, err)

	// The global codec used by ConsumeTxSizeGasDecorator, and private key
	// armoring (keys export), work after registration.
	RegisterEthKeysOnLegacyAmino()
	require.NotPanics(t, func() { legacy.Cdc.MustMarshal(sig) })
	require.NotPanics(t, func() { legacy.Cdc.MustMarshal(priv) })

	// Registering again, or on a codec that already knows the types, is a no-op.
	require.NotPanics(t, func() { RegisterEthKeysOnLegacyAmino(bare) })
	require.NotPanics(t, func() { RegisterEthKeysOnLegacyAmino() })

	// Existing encodings are unchanged: a secp256k1 signature encodes the same
	// on the bare codec before and after registration.
	plain := codec.NewLegacyAmino()
	cryptocodec.RegisterCrypto(plain)
	secp := legacytx.StdSignature{PubKey: simSecpPubKey(), Signature: make([]byte, 64)} //nolint:staticcheck
	want, err := plain.Marshal(secp)
	require.NoError(t, err)
	got, err := bare.Marshal(secp)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func simSecpPubKey() *secp256k1.PubKey {
	return secp256k1.GenPrivKey().PubKey().(*secp256k1.PubKey)
}
