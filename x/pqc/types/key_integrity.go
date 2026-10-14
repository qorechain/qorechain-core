package types

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Key integrity errors (v3.2.3). Codes from 230 keep clear of the ranges the
// other v3.2.x fixes use.
var (
	ErrWeakPQCKey                = errors.Register(ModuleName, 230, "PQC public key gives no post-quantum protection")
	ErrInvalidProofOfPossession  = errors.Register(ModuleName, 231, "PQC proof of possession does not verify")
	ErrPQCKeyReused              = errors.Register(ModuleName, 232, "PQC key was used by this account before")
	ErrLastSignatureAlgorithm    = errors.Register(ModuleName, 233, "no active PQC signature algorithm would remain")
	ErrPQCMaintenanceNeedsHybrid = errors.Register(ModuleName, 234, "PQC key maintenance from a keyed account needs the hybrid signature")
)

// mldsa87RhoSize is the size of rho, the seed of the public matrix, at the
// start of an ML-DSA public key. The rest of the key is t1.
const mldsa87RhoSize = 32

// IsDegeneratePublicKey reports whether pub is a key nobody needs a secret
// for. An ML-DSA public key is rho || t1, and every byte string of the right
// length decodes as one. When t1 is all zero the verification equation no
// longer depends on the challenge, so anyone can forge a signature for any
// message (QSR-2026-0538, proven against the fips204 crate the chain links).
// Such a key protects nothing. Only the algorithms this binary ships are
// judged; an unknown one returns false.
func IsDegeneratePublicKey(algo AlgorithmID, pub []byte) bool {
	if algo != AlgorithmDilithium5 && algo != AlgorithmUnspecified {
		return false
	}
	want, _ := ExpectedPublicKeySize(AlgorithmDilithium5)
	if len(pub) != int(want) {
		return false
	}
	for _, b := range pub[mldsa87RhoSize:] {
		if b != 0 {
			return false
		}
	}
	return true
}

// ValidateRegisteredKey is the registration integrity rule
// (FeaturePQCRegistrationIntegrity): a registration must carry a post-quantum
// key, and one that protects something. classical_only records a key the
// hybrid ante can never verify, which left the account unable to send
// anything again (QSR-2026-0550, 0400).
func ValidateRegisteredKey(keyType string, algo AlgorithmID, pub []byte) error {
	if keyType == KeyTypeClassicalOnly {
		return ErrWeakPQCKey.Wrap("key_type classical_only registers no post-quantum key; " +
			"the account would be unable to sign any transaction afterwards")
	}
	if len(pub) == 0 {
		return ErrWeakPQCKey.Wrap("a post-quantum public key is required")
	}
	if IsDegeneratePublicKey(algo, pub) {
		return ErrWeakPQCKey.Wrap("the key's t1 part is all zero, so anyone can forge its signatures")
	}
	return nil
}

// IsRepairableRecord reports whether a stored record holds no usable key, so
// a new MsgRegisterPQCKeyV2 may replace it. Such a record could only come
// from a classical_only or empty registration, which locked the account.
func IsRepairableRecord(acct PQCAccountInfo) bool {
	return len(acct.PublicKey) == 0 || acct.KeyType == KeyTypeClassicalOnly
}

// ProofOfPossessionSignBytes is what the key being registered signs to prove
// its owner holds the secret. It names the chain, the algorithm, the account
// and the key, so a proof made for one account or chain proves nothing for
// another.
func ProofOfPossessionSignBytes(chainID string, algorithmID uint32, account string, pub []byte) []byte {
	return []byte(fmt.Sprintf("qorechain-pqc-pop-v1|%s|%d|%s|%x", chainID, algorithmID, account, pub))
}

// RetiredKeyHash is how a key the account has rotated or migrated away from
// is remembered.
func RetiredKeyHash(pub []byte) []byte {
	h := sha256.Sum256(pub)
	return h[:]
}

// HasRetiredKey reports whether pub is a key this account used before.
func (a PQCAccountInfo) HasRetiredKey(pub []byte) bool {
	h := RetiredKeyHash(pub)
	for _, r := range a.RetiredKeyHashes {
		if bytes.Equal(r, h) {
			return true
		}
	}
	return false
}

// RetireKey remembers pub as a key this account may not return to.
func (a *PQCAccountInfo) RetireKey(pub []byte) {
	if len(pub) == 0 || a.HasRetiredKey(pub) {
		return
	}
	a.RetiredKeyHashes = append(a.RetiredKeyHashes, RetiredKeyHash(pub))
}

// PQCKeyManagementMsgTypeURLs are the messages that manage an account's own
// post-quantum key. None of them may be sent on an account's behalf through
// authz (FeaturePQCRotationReplayGuard).
func PQCKeyManagementMsgTypeURLs() []string {
	return []string{
		sdk.MsgTypeURL(&MsgRegisterPQCKey{}),
		sdk.MsgTypeURL(&MsgRegisterPQCKeyV2{}),
		sdk.MsgTypeURL(&MsgRotatePQCKey{}),
		sdk.MsgTypeURL(&MsgMigratePQCKey{}),
	}
}

// MempoolKeyCheck is the stateless key check a node applies to PQC key
// messages before admitting them to its mempool. It refuses what the
// registration integrity switch refuses in block execution, and keys of the
// wrong size in a rotation, so a transaction that can only fail is not
// relayed or charged. It is node-local: block execution never calls it.
func MempoolKeyCheck(msg sdk.Msg) error {
	switch m := msg.(type) {
	case *MsgRegisterPQCKey:
		return ValidateRegisteredKey(m.KeyType, AlgorithmDilithium5, m.DilithiumPubkey)
	case *MsgRegisterPQCKeyV2:
		return ValidateRegisteredKey(m.KeyType, m.AlgorithmID, m.PublicKey)
	case *MsgRotatePQCKey:
		// A rotation keeps the algorithm, so both keys have one size. The
		// size itself is the account's algorithm's, which needs state; a
		// governance-added algorithm may have another.
		if len(m.OldPublicKey) != len(m.NewPublicKey) {
			return ErrInvalidKeyLength.Wrapf("old and new keys of a rotation share an algorithm and a size, got %d and %d bytes",
				len(m.OldPublicKey), len(m.NewPublicKey))
		}
		if IsDegeneratePublicKey(AlgorithmDilithium5, m.NewPublicKey) {
			return ErrWeakPQCKey.Wrap("the new key's t1 part is all zero, so anyone can forge its signatures")
		}
	case *MsgMigratePQCKey:
		if IsDegeneratePublicKey(m.NewAlgorithmID, m.NewPublicKey) {
			return ErrWeakPQCKey.Wrap("the new key's t1 part is all zero, so anyone can forge its signatures")
		}
	}
	return nil
}
