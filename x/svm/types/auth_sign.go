package types

import "crypto/sha256"

// SVMAuthSignBytes is the canonical, domain-separated message a foreign-scheme
// wallet (e.g. Phantom ed25519) signs to authorize an SVM action for its
// canonical account:
//
//	sha256( "qorechain-svm-auth-v1" ‖ programID(32) ‖
//	        Σ[ addr(32) ‖ flags ] ‖ data ‖ recentBlockhash(32) )
//
// where flags = (isSigner?1) | (isWritable?2). Both the on-chain
// MsgExecuteProgram handler and the Solana-compatible sendTransaction RPC build
// it identically, so a signature verifies the same way through either path. The
// domain tag prevents cross-protocol signature reuse; binding the exact program,
// accounts, data and a recent blockhash prevents replay for a different action
// or (via the blockhash window) indefinitely.
func SVMAuthSignBytes(programID [32]byte, accounts []AccountMeta, data, recentBlockhash []byte) []byte {
	h := sha256.New()
	h.Write([]byte("qorechain-svm-auth-v1"))
	h.Write(programID[:])
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
	h.Write(data)
	h.Write(recentBlockhash)
	return h.Sum(nil)
}
