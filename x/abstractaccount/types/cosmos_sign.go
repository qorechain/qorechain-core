package types

import (
	"crypto/sha256"
	"encoding/binary"
)

// CosmosAuthSignBytes returns the domain-separated, replay-bound bytes an
// authenticator signs to authorize a MsgExecuteCosmos (v3.1.85). It binds the
// chain-id, the canonical account, the authenticator pubkey, the recipient, the
// amount and a per-authenticator sequence — each length-prefixed to remove
// concatenation ambiguity — so a signature cannot be replayed across chains,
// accounts, keys, recipients, amounts or sequences. Clients (the relayer /
// QoreX) build the identical bytes to sign. `to` is the exact bech32 string and
// `amount` is the canonical sdk.Coins string (sorted, e.g. "100uqor") carried in
// the message, so client and chain hash the same input. The domain prefix
// ("qorechain-cosmos-auth-v1") differs from the EVM domain, so an EVM-lane
// signature can never authorize a Native-lane spend and vice-versa.
func CosmosAuthSignBytes(chainID, account string, pubkey []byte, to, amount string, nonce uint64) []byte {
	b := make([]byte, 0, 64+len(pubkey)+len(amount))
	b = append(b, []byte("qorechain-cosmos-auth-v1")...)
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
	add([]byte(amount))
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], nonce)
	b = append(b, n[:]...)
	h := sha256.Sum256(b)
	return h[:]
}
