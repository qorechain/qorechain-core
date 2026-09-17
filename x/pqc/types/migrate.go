package types

import (
	"encoding/binary"
	"fmt"
)

// MigrationSignBytesDomain tags the message both keys sign in an algorithm
// migration. Versioned: v1 was
// "qorechain-key-migration:chain=..:from=..:to=..:account=..:height=..", which
// named the account and the algorithms but NOT the keys. The keeper verified
// the new signature under whatever new_public_key the message carried, so a
// migration signed once could be resubmitted with the destination key swapped
// for anyone's, and the account would end up registered to that key.
const MigrationSignBytesDomain = "qorechain-key-migration-v2"

// MigrationSignBytes is the message the old key and the new key both sign to
// migrate an account from one algorithm to another. Every field that decides
// what the chain will store is inside it, length-prefixed so no two field
// layouts encode the same bytes:
//
//	domain ‖ BE64(len chainID) ‖ chainID ‖ BE64(len account) ‖ account ‖
//	BE32(fromAlgo) ‖ BE32(toAlgo) ‖ BE64(height) ‖
//	BE32(len oldPub) ‖ oldPub ‖ BE32(len newPub) ‖ newPub
//
// Binding both public keys is the point: the new signature proving control of
// new_public_key is only meaningful if the OLD key also vouched for that exact
// destination. The execution height is kept from v1; it is what stops a
// migration being replayed later, and the client learns it from the chain.
// MigrationSignBytesLegacy is the first preimage, verified below the release
// that introduced v2 (protocol feature pqc-sign-bytes-v2). It names the
// account, the algorithms and the height, not the keys; kept verbatim so that
// history replays byte-identically.
func MigrationSignBytesLegacy(chainID, account string, fromAlgo, toAlgo uint32, height int64) []byte {
	return []byte(fmt.Sprintf(
		"qorechain-key-migration:chain=%s:from=%d:to=%d:account=%s:height=%d",
		chainID, fromAlgo, toAlgo, account, height,
	))
}

func MigrationSignBytes(chainID, account string, fromAlgo, toAlgo uint32, height int64, oldPub, newPub []byte) []byte {
	out := make([]byte, 0, len(MigrationSignBytesDomain)+8+len(chainID)+8+len(account)+4+4+8+4+len(oldPub)+4+len(newPub))
	out = append(out, MigrationSignBytesDomain...)
	out = binary.BigEndian.AppendUint64(out, uint64(len(chainID)))
	out = append(out, chainID...)
	out = binary.BigEndian.AppendUint64(out, uint64(len(account)))
	out = append(out, account...)
	out = binary.BigEndian.AppendUint32(out, fromAlgo)
	out = binary.BigEndian.AppendUint32(out, toAlgo)
	out = binary.BigEndian.AppendUint64(out, uint64(height))
	out = binary.BigEndian.AppendUint32(out, uint32(len(oldPub)))
	out = append(out, oldPub...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(newPub)))
	out = append(out, newPub...)
	return out
}
