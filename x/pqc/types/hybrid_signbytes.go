package types

import "encoding/binary"

// HybridSignBytesDomain is the domain tag on the post-quantum hybrid sign-bytes.
const HybridSignBytesDomain = "qorechain-pqc-hybrid-v2"

// HybridSignBytes is the message the ML-DSA key signs for a Cosmos-lane
// transaction. Both sides compute it: the wallet before signing, the chain before
// verifying.
//
//	"qorechain-pqc-hybrid-v2" ‖ BE64(len(chainID)) ‖ chainID ‖
//	  BE32(len(B0)) ‖ B0 ‖ BE32(len(authInfo)) ‖ authInfo
//
// where B0 is the TxBody serialised WITHOUT the PQC extension option (so the
// signature does not have to cover the bytes that carry it) and authInfo is the
// AuthInfo bytes verbatim.
//
// WHY THE DOMAIN TAG AND THE CHAIN-ID WERE ADDED. The previous form was the two
// length-prefixed fields alone. That was injective, so no two transactions shared
// sign-bytes, but it bound nothing outside the transaction: the same key signs
// other things in a wallet's life, a login challenge, a message-signing request,
// and none of those carried a tag that made them distinguishable from transaction
// sign-bytes. A signature obtained in one of those contexts could be valid here.
// The chain-id is bound as well, so the post-quantum signature itself refuses to
// verify on another network, rather than relying on the classical signature in
// the same transaction to do that job.
//
// Changing this changes the wallet-to-chain contract. The reference cosign
// implementation in x/pqc/client/cli, the public SDK and the wallet adapter all
// build these bytes and must move together; a client on the old form produces a
// digest this chain rejects, which fails closed rather than ambiguously.
// SignBytesV2Upgrade names the coordinated upgrade whose handler switches a
// chain that existed before it from the first sign-bytes form to v2. A client
// asks the chain whether that plan has been applied to know which form to
// sign; the chain itself decides by its protocol-feature record.
const SignBytesV2Upgrade = "v3.1.98"

// legacySignBytesChains are the networks that were running before v2 existed
// and switch to it only at SignBytesV2Upgrade. Any other chain starts on a
// binary that verifies v2 from its first block.
var legacySignBytesChains = map[string]bool{
	"qorechain-vladi": true,
	"qorechain-diana": true,
}

// SignBytesVersionFor decides the form a client must sign for chainID, given
// the height at which SignBytesV2Upgrade was applied on that chain (0 when it
// has not been). It is the client-side mirror of the chain's switch.
func SignBytesVersionFor(chainID string, v2AppliedHeight int64) int {
	if v2AppliedHeight > 0 || !legacySignBytesChains[chainID] {
		return 2
	}
	return 1
}

// HybridSignBytesLegacy is the first form, verified for every block below the
// release that introduced v2 (protocol feature pqc-sign-bytes-v2), so that
// history replays byte-identically:
//
//	BE32(len(B0)) ‖ B0 ‖ BE32(len(authInfo)) ‖ authInfo
func HybridSignBytesLegacy(b0, authInfo []byte) []byte {
	buf := make([]byte, 4+len(b0)+4+len(authInfo))
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(b0)))
	copy(buf[4:4+len(b0)], b0)
	binary.BigEndian.PutUint32(buf[4+len(b0):8+len(b0)], uint32(len(authInfo)))
	copy(buf[8+len(b0):], authInfo)
	return buf
}

func HybridSignBytes(chainID string, b0, authInfo []byte) []byte {
	out := make([]byte, 0, len(HybridSignBytesDomain)+8+len(chainID)+4+len(b0)+4+len(authInfo))
	out = append(out, HybridSignBytesDomain...)
	var n64 [8]byte
	binary.BigEndian.PutUint64(n64[:], uint64(len(chainID)))
	out = append(out, n64[:]...)
	out = append(out, chainID...)
	var n32 [4]byte
	binary.BigEndian.PutUint32(n32[:], uint32(len(b0)))
	out = append(out, n32[:]...)
	out = append(out, b0...)
	binary.BigEndian.PutUint32(n32[:], uint32(len(authInfo)))
	out = append(out, n32[:]...)
	out = append(out, authInfo...)
	return out
}
