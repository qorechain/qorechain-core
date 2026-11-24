package types

import (
	"encoding/hex"
	"fmt"
)

// AuthenticatorLinkDomain names the message a key signs to prove it agrees to
// be linked to an account (abstractaccount-authenticator-possession).
const AuthenticatorLinkDomain = "QoreChain authenticator link v1"

// AuthenticatorLinkMessage is what the key being linked signs, under its own
// scheme, to put in MsgRegisterAuthenticator.possession_proof. It binds the
// chain, the account the key is linked to, the scheme and the key itself, so
// a proof cannot be replayed on another chain, for another account or for
// another key. It is plain text so that a browser wallet shows the user what
// they sign (Phantom signMessage, MetaMask personal_sign); for a 33-byte
// secp256k1 key the signature is over sha256 of these bytes, as for any
// Cosmos signature.
//
//	QoreChain authenticator link v1
//	chain: <chain-id>
//	account: <bech32 account>
//	scheme: <scheme>
//	key: <lowercase hex of the public key, or of the 20-byte address>
func AuthenticatorLinkMessage(chainID, account, scheme string, pubkey []byte) []byte {
	return []byte(fmt.Sprintf("%s\nchain: %s\naccount: %s\nscheme: %s\nkey: %s",
		AuthenticatorLinkDomain, chainID, account, scheme, hex.EncodeToString(pubkey)))
}
