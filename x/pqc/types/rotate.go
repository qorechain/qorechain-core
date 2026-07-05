package types

import "fmt"

// RotationSignBytes returns the domain-separated bytes that BOTH the old and the
// new key sign for a MsgRotatePQCKey. It binds the chain-id, algorithm, account,
// and both public keys. There is deliberately NO block height (the signer cannot
// predict the execution height); replay is prevented structurally — after the
// rotation the old key no longer matches the registered key, so the same message
// cannot be replayed. The chain re-derives these exact bytes in the handler.
func RotationSignBytes(chainID string, algorithmID uint32, account string, oldPub, newPub []byte) []byte {
	return []byte(fmt.Sprintf("qorechain-pqc-rotate-v1|%s|%d|%s|%x|%x", chainID, algorithmID, account, oldPub, newPub))
}
