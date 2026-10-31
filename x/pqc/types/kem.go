package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Native KEM key registry (v3.2.5, FeaturePQCKEMRegistry).
//
// An account publishes one key-encapsulation public key so that anyone can
// encrypt to it: a sender reads the key from the chain, encapsulates a shared
// secret to it, and only the holder of the matching secret key can recover
// that secret. The key is NOT an authentication key. The account keeps signing
// with its ML-DSA key, and that signature is what authenticates the
// registration, so the published key needs no proof of possession of its own:
// a KEM cannot sign, and an account that publishes a key it cannot decapsulate
// with only stops itself from receiving.

// Errors of the KEM registry. Codes from 250 keep clear of the ranges the
// other v3.2.x fixes use.
var (
	ErrKEMRegistryNotEnabled = errors.Register(ModuleName, 250, "the KEM key registry is not enabled on this chain yet")
	ErrInvalidKEMKey         = errors.Register(ModuleName, 251, "invalid KEM public key")
	ErrKEMKeyNotFound        = errors.Register(ModuleName, 252, "no KEM key published for this account")
	ErrKEMKeyUnchanged       = errors.Register(ModuleName, 253, "this KEM key is already published for this account")
	ErrKEMNeedsSignatureKey  = errors.Register(ModuleName, 254, "publishing a KEM key needs a registered post-quantum signature key")
)

// MaxKEMPublicKeySize bounds the key of an algorithm this binary does not
// know, before the keeper checks it against the registry. ML-KEM-1024, the
// largest standard parameter set, uses 1568 bytes.
const MaxKEMPublicKeySize = 4096

// KEMKeyPrefix is the KV prefix for published KEM keys, one per account.
var KEMKeyPrefix = []byte("pqc/kem/")

// KEMKeyKey returns the store key for an account's KEM key.
func KEMKeyKey(addr []byte) []byte {
	return append(append([]byte{}, KEMKeyPrefix...), addr...)
}

// KEMKeyInfo is an account's published KEM key as stored.
type KEMKeyInfo struct {
	Address          string      `json:"address"`
	AlgorithmID      AlgorithmID `json:"algorithm_id"`
	PublicKey        []byte      `json:"public_key"`
	RegisteredHeight int64       `json:"registered_height"`
}

// ML-KEM-1024 parameters (FIPS 203, table 2): k = 4, q = 3329. The
// encapsulation key is ByteEncode_12(t_hat) || rho, 384*k + 32 bytes.
const (
	mlkemQ             = 3329
	mlkem1024K         = 4
	mlkemPolyBytes     = 384
	mlkemRhoSize       = 32
	mlkem1024THatBytes = mlkem1024K * mlkemPolyBytes
)

// ValidateMLKEM1024EncapsulationKey applies the FIPS 203 encapsulation key
// check (section 7.2, "modulus check"): every 12-bit coefficient of t_hat must
// be below q, which is what decoding and re-encoding the key leaves unchanged.
// A conforming encapsulator refuses a key that fails it, so a published key
// that fails it can never be encrypted to. It also refuses a key whose t_hat
// is all zero: encapsulation to it hides nothing, because the ciphertext's v
// part is then the message plus small noise, readable by anyone.
func ValidateMLKEM1024EncapsulationKey(pub []byte) error {
	want, _ := ExpectedPublicKeySize(AlgorithmMLKEM1024)
	if len(pub) != int(want) {
		return ErrInvalidKEMKey.Wrapf("an ML-KEM-1024 encapsulation key is %d bytes, got %d", want, len(pub))
	}
	zero := true
	for i := 0; i < mlkem1024THatBytes; i += 3 {
		c0 := uint16(pub[i]) | uint16(pub[i+1]&0x0f)<<8
		c1 := uint16(pub[i+1]>>4) | uint16(pub[i+2])<<4
		if c0 >= mlkemQ || c1 >= mlkemQ {
			return ErrInvalidKEMKey.Wrapf("coefficient at byte %d is not reduced modulo %d; the key fails the FIPS 203 encapsulation key check", i, mlkemQ)
		}
		if c0 != 0 || c1 != 0 {
			zero = false
		}
	}
	if zero {
		return ErrInvalidKEMKey.Wrap("the key's t_hat part is all zero, so anything encrypted to it is readable by anyone")
	}
	return nil
}

// ValidateKEMPublicKey is the stateless key rule: the algorithm must not be a
// known signature scheme, and a key of an algorithm this binary knows must
// have its size and pass its encoding check. An algorithm added by governance
// later is checked by the keeper against the on-chain registry instead.
func ValidateKEMPublicKey(algo AlgorithmID, pub []byte) error {
	if algo == AlgorithmUnspecified {
		return ErrInvalidAlgorithm.Wrap("algorithm_id is required")
	}
	if algo.IsSignature() {
		return ErrInvalidAlgorithm.Wrapf("%s is a signature algorithm, not a key-encapsulation mechanism; "+
			"signature keys are registered with register-key-v2", algo)
	}
	if len(pub) == 0 {
		return ErrInvalidKEMKey.Wrap("public_key is required")
	}
	if len(pub) > MaxKEMPublicKeySize {
		return ErrInvalidKEMKey.Wrapf("public key is %d bytes, more than the %d any KEM uses", len(pub), MaxKEMPublicKeySize)
	}
	if algo == AlgorithmMLKEM1024 {
		return ValidateMLKEM1024EncapsulationKey(pub)
	}
	return nil
}

// ValidateBasic for MsgRegisterKEMKey. The message type is new in v3.2.5, so
// no historical block can carry it and this check needs no switch.
func (msg *MsgRegisterKEMKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	return ValidateKEMPublicKey(msg.AlgorithmID, msg.PublicKey)
}

// ValidateBasic for MsgRemoveKEMKey.
func (msg *MsgRemoveKEMKey) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	return err
}

// IsKEMKeyMsg reports whether msg maintains an account's KEM key.
func IsKEMKeyMsg(msg sdk.Msg) bool {
	switch msg.(type) {
	case *MsgRegisterKEMKey, *MsgRemoveKEMKey:
		return true
	}
	return false
}

// ValidateRotationKeySizes is the stateless size rule of a rotation
// (QSR-2026-0607): a rotation keeps the algorithm, so the new key has the old
// key's length. Which length that is depends on the account's algorithm, which
// needs state; the handler checks it. MsgRotatePQCKey.ValidateBasic checks
// presence only and cannot be tightened (it runs in block execution and has
// no height); from FeaturePQCRotateKeySizeCheck this rule runs in the ante
// instead, before the fee.
func ValidateRotationKeySizes(msg *MsgRotatePQCKey) error {
	if len(msg.OldPublicKey) != len(msg.NewPublicKey) {
		return ErrInvalidKeyLength.Wrapf("old and new keys of a rotation share an algorithm and a size, got %d and %d bytes",
			len(msg.OldPublicKey), len(msg.NewPublicKey))
	}
	return nil
}

// FeatureGatedValidateBasic is the part of a message's stateless validation
// that reaches block execution only from a protocol switch. enabled reports
// whether a switch is on at the current height. Messages without such a rule
// pass.
func FeatureGatedValidateBasic(msg sdk.Msg, enabled func(string) bool) error {
	switch m := msg.(type) {
	case *MsgRotatePQCKey:
		if enabled(FeaturePQCRotateKeySizeCheck) {
			return ValidateRotationKeySizes(m)
		}
	}
	return nil
}
