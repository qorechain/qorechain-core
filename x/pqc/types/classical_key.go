package types

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// CompressedSecp256k1Size is the length of a compressed secp256k1 public key.
const CompressedSecp256k1Size = 33

// ValidateClassicalKey checks the ecdsa_pubkey a PQC key registration carries
// (QSR-2026-0281). The field is optional: empty is accepted and means "not
// given". A non-empty value must be a 33-byte compressed secp256k1 point that
// belongs to the signer, under either address derivation the chain uses:
// eth_secp256k1 (keccak, the unified qor1/0x account) or Cosmos secp256k1
// (sha256 + ripemd160).
//
// Before this the value was stored as sent, so the queried record could show a
// hybrid key type next to an empty, junk or somebody else's classical key.
// Nothing on chain verifies against it (classical signatures use the auth
// account's key), but explorers and SDKs display it.
func ValidateClassicalKey(sender string, pub []byte) error {
	if len(pub) == 0 {
		return nil
	}
	if len(pub) != CompressedSecp256k1Size || (pub[0] != 0x02 && pub[0] != 0x03) {
		return ErrInvalidClassicalKey.Wrapf("want a %d-byte compressed secp256k1 key, got %d bytes", CompressedSecp256k1Size, len(pub))
	}
	point, err := ethcrypto.DecompressPubkey(pub)
	if err != nil {
		return ErrInvalidClassicalKey.Wrapf("not a secp256k1 point: %v", err)
	}
	signer, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return ErrInvalidClassicalKey.Wrapf("invalid sender: %v", err)
	}
	if bytes.Equal(ethcrypto.PubkeyToAddress(*point).Bytes(), signer) {
		return nil
	}
	if bytes.Equal((&secp256k1.PubKey{Key: pub}).Address(), signer) {
		return nil
	}
	return ErrInvalidClassicalKey.Wrapf("key does not belong to %s", sender)
}
