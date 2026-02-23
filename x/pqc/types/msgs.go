package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------------------------------------------------------------------------
// MsgRegisterPQCKey — legacy v1 message (deprecated, still works)
// ---------------------------------------------------------------------------

// MsgRegisterPQCKey registers a PQC keypair for an account.
// Deprecated: Use MsgRegisterPQCKeyV2 which includes AlgorithmID.
// This message defaults to Dilithium-5 for backward compatibility.
type MsgRegisterPQCKey struct {
	Sender          string `json:"sender"`
	DilithiumPubkey []byte `json:"dilithium_pubkey"`
	ECDSAPubkey     []byte `json:"ecdsa_pubkey,omitempty"`
	KeyType         string `json:"key_type"`
}

func (msg *MsgRegisterPQCKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if len(msg.DilithiumPubkey) == 0 && msg.KeyType != KeyTypeClassicalOnly {
		return ErrInvalidKeyLength.Wrap("dilithium pubkey required for hybrid or pqc_only key type")
	}
	switch msg.KeyType {
	case KeyTypeHybrid, KeyTypePQCOnly, KeyTypeClassicalOnly:
		// valid
	default:
		return ErrInvalidKeyLength.Wrapf("invalid key type: %s", msg.KeyType)
	}
	return nil
}

func (msg *MsgRegisterPQCKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterPQCKey) ProtoMessage() {}
func (msg *MsgRegisterPQCKey) Reset()        {}
func (msg *MsgRegisterPQCKey) String() string { return "MsgRegisterPQCKey" }

// ---------------------------------------------------------------------------
// MsgRegisterPQCKeyV2 — algorithm-aware key registration (v0.6.0)
// ---------------------------------------------------------------------------

// MsgRegisterPQCKeyV2 registers a PQC key with explicit algorithm selection.
type MsgRegisterPQCKeyV2 struct {
	Sender      string      `json:"sender"`
	PublicKey   []byte      `json:"public_key"`
	AlgorithmID AlgorithmID `json:"algorithm_id"`
	ECDSAPubkey []byte      `json:"ecdsa_pubkey,omitempty"` // For hybrid mode
	KeyType     string      `json:"key_type"`
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
		// valid
	default:
		return ErrInvalidKeyLength.Wrapf("invalid key type: %s", msg.KeyType)
	}
	return nil
}

func (msg *MsgRegisterPQCKeyV2) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgRegisterPQCKeyV2) ProtoMessage() {}
func (msg *MsgRegisterPQCKeyV2) Reset()        {}
func (msg *MsgRegisterPQCKeyV2) String() string { return "MsgRegisterPQCKeyV2" }

// ---------------------------------------------------------------------------
// MsgMigratePQCKey — dual-signature key migration (v0.6.0)
// ---------------------------------------------------------------------------

// MsgMigratePQCKey migrates an account's PQC key from one algorithm to another.
// Requires dual signatures proving ownership of both keys.
type MsgMigratePQCKey struct {
	Sender         string      `json:"sender"`
	OldPublicKey   []byte      `json:"old_public_key"`
	NewPublicKey   []byte      `json:"new_public_key"`
	NewAlgorithmID AlgorithmID `json:"new_algorithm_id"`
	OldSignature   []byte      `json:"old_signature"` // Proves ownership of old key
	NewSignature   []byte      `json:"new_signature"` // Proves ownership of new key
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

func (msg *MsgMigratePQCKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgMigratePQCKey) ProtoMessage() {}
func (msg *MsgMigratePQCKey) Reset()        {}
func (msg *MsgMigratePQCKey) String() string { return "MsgMigratePQCKey" }

// ---------------------------------------------------------------------------
// Governance proposal messages (v0.6.0 — SDK v1 style: sdk.Msg)
// ---------------------------------------------------------------------------

// MsgAddAlgorithm proposes adding a new PQC algorithm to the registry.
// Must be submitted through governance (MsgSubmitProposal).
type MsgAddAlgorithm struct {
	Authority string        `json:"authority"` // Governance module address
	Algorithm AlgorithmInfo `json:"algorithm"`
}

func (msg *MsgAddAlgorithm) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return err
	}
	return msg.Algorithm.Validate()
}

func (msg *MsgAddAlgorithm) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddAlgorithm) ProtoMessage() {}
func (msg *MsgAddAlgorithm) Reset()        {}
func (msg *MsgAddAlgorithm) String() string { return "MsgAddAlgorithm" }

// MsgDeprecateAlgorithm proposes deprecating an algorithm (starts migration period).
type MsgDeprecateAlgorithm struct {
	Authority        string      `json:"authority"`
	AlgorithmID      AlgorithmID `json:"algorithm_id"`
	MigrationBlocks  int64       `json:"migration_blocks"` // Dual-sig period in blocks
	ReplacementAlgID AlgorithmID `json:"replacement_algorithm_id"`
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

func (msg *MsgDeprecateAlgorithm) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDeprecateAlgorithm) ProtoMessage() {}
func (msg *MsgDeprecateAlgorithm) Reset()        {}
func (msg *MsgDeprecateAlgorithm) String() string { return "MsgDeprecateAlgorithm" }

// MsgDisableAlgorithm emergency-disables an algorithm (e.g., vulnerability discovered).
type MsgDisableAlgorithm struct {
	Authority   string      `json:"authority"`
	AlgorithmID AlgorithmID `json:"algorithm_id"`
	Reason      string      `json:"reason"`
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

func (msg *MsgDisableAlgorithm) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgDisableAlgorithm) ProtoMessage() {}
func (msg *MsgDisableAlgorithm) Reset()        {}
func (msg *MsgDisableAlgorithm) String() string { return "MsgDisableAlgorithm" }
