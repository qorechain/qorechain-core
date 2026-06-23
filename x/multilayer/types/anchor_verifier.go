package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AnchorSignatureVerifier authenticates a state anchor's PQC attestation: it
// resolves the attestor's registered Dilithium public key and verifies the
// signature over the canonical anchor message. Defined in types so both the
// multilayer package and its keeper can reference it without an import cycle.
type AnchorSignatureVerifier interface {
	// AttestorPubKey returns the registered Dilithium public key for a bech32
	// address, and whether one is registered.
	AttestorPubKey(ctx sdk.Context, addr string) (pubKey []byte, found bool)
	// VerifyDilithium verifies a Dilithium-5 signature over message.
	VerifyDilithium(ctx sdk.Context, pubKey, message, signature []byte) (bool, error)
}
