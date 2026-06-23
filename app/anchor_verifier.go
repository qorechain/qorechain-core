package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// pqcAnchorVerifier authenticates multilayer state anchors against the layer
// creator's registered Dilithium key, using the x/pqc module. It implements
// multilayer.AnchorSignatureVerifier.
type pqcAnchorVerifier struct {
	pqc pqcmod.PQCKeeper
}

// AttestorPubKey returns the creator's registered Dilithium-5 public key, if any.
func (v pqcAnchorVerifier) AttestorPubKey(ctx sdk.Context, addr string) ([]byte, bool) {
	info, found := v.pqc.GetPQCAccount(ctx, addr)
	if !found || info.AlgorithmID != pqctypes.AlgorithmDilithium5 || len(info.PublicKey) == 0 {
		return nil, false
	}
	return info.PublicKey, true
}

// VerifyDilithium verifies a Dilithium-5 signature via the PQC client.
func (v pqcAnchorVerifier) VerifyDilithium(_ sdk.Context, pubKey, message, signature []byte) (bool, error) {
	return v.pqc.PQCClient().DilithiumVerify(pubKey, message, signature)
}
