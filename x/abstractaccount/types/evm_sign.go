package types

import (
	"crypto/sha256"
	"encoding/binary"
)

// EVMAuthSignBytes returns the domain-separated, replay-bound bytes an
// authenticator signs to authorize a MsgExecuteEVM (v3.1.85). It binds the
// chain-id, the canonical account, the authenticator pubkey, the EVM call
// (to/value/data) and the account's expected EVM nonce — each length-prefixed to
// remove concatenation ambiguity — so a signature cannot be replayed across
// chains, accounts, keys, calls or nonces. Clients (the relayer / QoreX) build
// the identical bytes to sign. `to` and `value` are the exact strings carried in
// the message (0x-hex address, decimal wei) so client and chain hash the same input.
func EVMAuthSignBytes(chainID, account string, pubkey []byte, to, value string, data []byte, nonce uint64) []byte {
	b := make([]byte, 0, 64+len(pubkey)+len(data))
	b = append(b, []byte("qorechain-evm-auth-v1")...)
	add := func(x []byte) {
		var l [8]byte
		binary.BigEndian.PutUint64(l[:], uint64(len(x)))
		b = append(b, l[:]...)
		b = append(b, x...)
	}
	add([]byte(chainID))
	add([]byte(account))
	add(pubkey)
	add([]byte(to))
	add([]byte(value))
	add(data)
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], nonce)
	b = append(b, n[:]...)
	h := sha256.Sum256(b)
	return h[:]
}
