package types

import "fmt"

// RotationSignBytes returns the domain-separated bytes that BOTH the old and the
// new key sign for a MsgRotatePQCKey. It binds the chain-id, algorithm, account,
// and both public keys. There is deliberately NO block height (the signer cannot
// predict the execution height) and no nonce. After the rotation the old key no
// longer matches the registered key, but that alone does NOT prevent replay: once
// the account cycles back (A->B->A) the original A->B message verifies again
// (QSR-2026-0506, 0372). From v3.2.3 (FeaturePQCRotationReplayGuard) the handler
// refuses any rotation or migration to a key the account used before, which
// closes the replay without changing these bytes. The chain re-derives these
// exact bytes in the handler.
func RotationSignBytes(chainID string, algorithmID uint32, account string, oldPub, newPub []byte) []byte {
	return []byte(fmt.Sprintf("qorechain-pqc-rotate-v1|%s|%d|%s|%x|%x", chainID, algorithmID, account, oldPub, newPub))
}
