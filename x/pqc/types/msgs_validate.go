package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implementations for the proto-generated PQC messages.
// (The message structs themselves are generated in tx.pb.go.)

func (msg *MsgRegisterPQCKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if len(msg.DilithiumPubkey) == 0 && msg.KeyType != KeyTypeClassicalOnly {
		return ErrInvalidKeyLength.Wrap("dilithium pubkey required for hybrid or pqc_only key type")
	}
	// THE LENGTH IS CHECKED, NOT JUST THE PRESENCE.
	//
	// Only non-emptiness was checked here, and the 2592-byte size check lives
	// elsewhere, applied to keys carried inside a hybrid signature rather than to a
	// key being registered. Under the post-quantum-required posture this chain runs,
	// an account that registers a short or malformed key can never produce a valid
	// signature again, for anything, including the rotation that would repair it:
	// both rotate and migrate verify against the key already on file. One
	// transaction, self-inflicted, irreversible, and nothing stopped it.
	if len(msg.DilithiumPubkey) > 0 {
		if want := int(DefaultDilithium5Info().PublicKeySize); len(msg.DilithiumPubkey) != want {
			return ErrInvalidKeyLength.Wrapf(
				"dilithium public key must be %d bytes, got %d; registering a key of the "+
					"wrong length leaves the account unable to sign anything ever again, "+
					"including the rotation that would undo it", want, len(msg.DilithiumPubkey))
		}
	}
	switch msg.KeyType {
	case KeyTypeHybrid, KeyTypePQCOnly, KeyTypeClassicalOnly:
		return nil
	default:
		return ErrInvalidKeyLength.Wrapf("invalid key type: %s", msg.KeyType)
	}
}

func (msg *MsgRegisterPQCKeyV2) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if msg.AlgorithmID == AlgorithmUnspecified {
		return ErrInvalidAlgorithm.Wrap("algorithm_id is required")
	}
	// An authentication key must be able to sign. ML-KEM is a key-encapsulation
	// mechanism with no signing operation, so registering it leaves the account
	// unable to produce the hybrid signature the chain requires, and unable to
	// rotate out of that state because rotation needs a valid signature under the
	// key on file.
	//
	// This rejects what is STATICALLY known to be a KEM rather than demanding
	// IsSignature(). The difference matters: IsSignature() is a fixed switch over
	// the built-ins, so requiring it here would also reject a signature algorithm
	// added later by governance, which ValidateBasic cannot know about because it
	// has no access to the on-chain algorithm registry. The keeper performs the
	// positive check against that registry; this is the cheap, early, always-safe
	// half.
	if msg.AlgorithmID.IsKEM() {
		return ErrInvalidAlgorithm.Wrapf(
			"algorithm %s is a key-encapsulation mechanism and cannot be used as an "+
				"authentication key", msg.AlgorithmID)
	}
	if len(msg.PublicKey) == 0 && msg.KeyType != KeyTypeClassicalOnly {
		return ErrInvalidKeyLength.Wrap("public_key required for hybrid or pqc_only key type")
	}
	// Length is checked for the algorithms whose size is known at compile time, for
	// the same reason the KEM check above is: a key of the wrong length leaves the
	// account unable to sign anything ever again, including the rotation that would
	// repair it, because rotate and migrate both verify against the key on file.
	//
	// An algorithm governance added after this binary shipped is not checked here,
	// only in the keeper against the on-chain registry. That is the same split as the
	// KEM check: refuse what is statically known to be wrong, and leave what needs
	// state to the place that has it.
	if want, known := ExpectedPublicKeySize(msg.AlgorithmID); known && len(msg.PublicKey) > 0 {
		if len(msg.PublicKey) != int(want) {
			return ErrInvalidKeyLength.Wrapf(
				"a %s public key must be %d bytes, got %d; registering a key of the wrong "+
					"length leaves the account permanently unable to sign, with no way back",
				msg.AlgorithmID, want, len(msg.PublicKey))
		}
	}
	switch msg.KeyType {
	case KeyTypeHybrid, KeyTypePQCOnly, KeyTypeClassicalOnly:
		return nil
	default:
		return ErrInvalidKeyLength.Wrapf("invalid key type: %s", msg.KeyType)
	}
}

func (msg *MsgMigratePQCKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if len(msg.OldPublicKey) == 0 {
		return ErrInvalidKeyLength.Wrap("old_public_key is required")
	}
	if len(msg.NewPublicKey) == 0 {
		return ErrInvalidKeyLength.Wrap("new_public_key is required")
	}
	if msg.NewAlgorithmID == AlgorithmUnspecified {
		return ErrInvalidAlgorithm.Wrap("new_algorithm_id is required")
	}
	if len(msg.OldSignature) == 0 {
		return ErrDualSigRequired.Wrap("old_signature is required for migration")
	}
	if len(msg.NewSignature) == 0 {
		return ErrDualSigRequired.Wrap("new_signature is required for migration")
	}
	return nil
}

func (msg *MsgRotatePQCKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if len(msg.OldPublicKey) == 0 {
		return ErrInvalidKeyLength.Wrap("old_public_key is required")
	}
	if len(msg.NewPublicKey) == 0 {
		return ErrInvalidKeyLength.Wrap("new_public_key is required")
	}
	if len(msg.OldSignature) == 0 {
		return ErrDualSigRequired.Wrap("old_signature is required for rotation")
	}
	if len(msg.NewSignature) == 0 {
		return ErrDualSigRequired.Wrap("new_signature is required for rotation")
	}
	return nil
}

func (msg *MsgDeprecateAlgorithm) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return err
	}
	if msg.AlgorithmID == AlgorithmUnspecified {
		return ErrInvalidAlgorithm.Wrap("algorithm_id is required")
	}
	if msg.ReplacementAlgID == AlgorithmUnspecified {
		return ErrInvalidAlgorithm.Wrap("replacement_algorithm_id is required")
	}
	if msg.AlgorithmID == msg.ReplacementAlgID {
		return ErrInvalidAlgorithm.Wrap("cannot migrate an algorithm to itself")
	}
	if msg.MigrationBlocks <= 0 {
		return ErrInvalidAlgorithm.Wrap("migration_blocks must be positive")
	}
	return nil
}

func (msg *MsgDisableAlgorithm) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return err
	}
	if msg.AlgorithmID == AlgorithmUnspecified {
		return ErrInvalidAlgorithm.Wrap("algorithm_id is required")
	}
	if msg.Reason == "" {
		return ErrInvalidAlgorithm.Wrap("reason is required for emergency disable")
	}
	return nil
}
