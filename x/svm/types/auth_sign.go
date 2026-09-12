package types

import (
	"crypto/sha256"
	"encoding/binary"
)

// SVMAuthSignBytes is the canonical, domain-separated message a foreign-scheme
// wallet key (Phantom ed25519, MetaMask secp256k1) signs to authorise ONE SVM
// instruction on behalf of the account its authenticator is bound to.
//
//	sha256( "qorechain-svm-auth-v2" ‖ programID(32) ‖
//	        BE64(len(accounts)) ‖ Σ[ addr(32) ‖ flags(1) ] ‖
//	        BE64(len(data)) ‖ data ‖
//	        BE64(len(recentBlockhash)) ‖ recentBlockhash )
//
// where flags = (isSigner?1) | (isWritable?2).
//
// WHY v2 AND WHY THE LENGTH PREFIXES. The v1 encoding concatenated the account
// list, the instruction data and the blockhash with no count and no length
// framing. That made the encoding NON-INJECTIVE, so one signature could be valid
// for more than one logical instruction:
//
//   - account entries are a fixed 33 bytes each and `data` followed them
//     directly, so N accounts plus data D hashed identically to N-1 accounts plus
//     data (addr‖flags‖D). A signature the holder approved for one account set
//     therefore also authorised a DIFFERENT account set, including different
//     writable and signer flags.
//   - `data` and `recentBlockhash` were adjacent and both variable length, so
//     bytes could be moved from the end of one into the front of the other.
//
// The count prefix fixes the list boundary and the length prefixes make every
// variable field self-delimiting, so distinct instructions always hash
// differently. The domain tag is bumped to v2 deliberately: a client still
// building v1 bytes now produces a digest the node will not accept, which fails
// closed instead of signing something ambiguous. The SVM lane was compile-time
// disabled when this changed and the authenticator path had never been used, so
// no signature in existence depends on v1.
//
// Any client that rebuilds this digest must match it byte for byte. The
// reference JavaScript implementation lives in qorechain-wallet-adapter
// (src/phantom.js, authSignBytes).
//
// The big-endian uint64 length prefix is deliberate rather than arbitrary: it is
// the convention the other two sign-bytes helpers in this codebase already use
// (x/abstractaccount/types/cosmos_sign.go and evm_sign.go). One framing
// convention across every signed payload is worth more than a shorter prefix
// here, because the next person writing one copies whatever they find.
func SVMAuthSignBytes(programID [32]byte, accounts []AccountMeta, data, recentBlockhash []byte) []byte {
	h := sha256.New()
	h.Write([]byte("qorechain-svm-auth-v2"))
	h.Write(programID[:])

	writeBE64(h, len(accounts))
	for _, m := range accounts {
		h.Write(m.Address[:])
		var flags byte
		if m.IsSigner {
			flags |= 1
		}
		if m.IsWritable {
			flags |= 2
		}
		h.Write([]byte{flags})
	}

	writeLenPrefixed(h, data)
	writeLenPrefixed(h, recentBlockhash)
	return h.Sum(nil)
}

// writeBE64 writes n as a big-endian uint64, matching the framing used by the
// other sign-bytes helpers in this codebase.
func writeBE64(h interface{ Write([]byte) (int, error) }, n int) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(n))
	h.Write(b[:])
}

// writeLenPrefixed writes BE64(len(p)) followed by p, so the field is
// self-delimiting and cannot borrow bytes from its neighbour.
func writeLenPrefixed(h interface{ Write([]byte) (int, error) }, p []byte) {
	writeBE64(h, len(p))
	h.Write(p)
}
