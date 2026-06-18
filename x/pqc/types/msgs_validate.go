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
	if len(msg.PublicKey) == 0 && msg.KeyType != KeyTypeClassicalOnly {
		return ErrInvalidKeyLength.Wrap("public_key required for hybrid or pqc_only key type")
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
