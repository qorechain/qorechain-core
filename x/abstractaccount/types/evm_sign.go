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
	return evmAuthSignBytes("qorechain-evm-auth-v1", chainID, account, pubkey, to, value, data, nonce, nil)
}

// EVMAuthSignBytesV2 is EVMAuthSignBytes under its own domain tag with the
// call's gas limit appended as a big-endian uint64 (QSR-2026-0474). The v1
// digest left gas_limit to whoever relayed the message, so the relayer could
// replay a valid signature with a different limit and change the outcome of a
// gas-sensitive call (an inner call that runs out of gas, a gasleft() branch,
// or a failed attempt followed by a retry under the same nonce). The chain
// requires this form from the abstractaccount-hardening protocol feature on;
// below it, only v1. No block accepts both.
func EVMAuthSignBytesV2(chainID, account string, pubkey []byte, to, value string, data []byte, nonce, gasLimit uint64) []byte {
	var g [8]byte
	binary.BigEndian.PutUint64(g[:], gasLimit)
	return evmAuthSignBytes("qorechain-evm-auth-v2", chainID, account, pubkey, to, value, data, nonce, g[:])
}

func evmAuthSignBytes(tag, chainID, account string, pubkey []byte, to, value string, data []byte, nonce uint64, tail []byte) []byte {
	b := make([]byte, 0, 72+len(pubkey)+len(data))
	b = append(b, []byte(tag)...)
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
	b = append(b, tail...)
	h := sha256.Sum256(b)
	return h[:]
}
