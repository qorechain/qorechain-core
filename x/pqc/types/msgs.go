package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgRegisterPQCKey registers a PQC keypair for an account.
// Note: This is a plain Go struct for now. When protobuf definitions are added,
// this will become a proto-generated type implementing sdk.Msg.
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

// Proto stubs to satisfy sdk.Msg interface (until protobuf is generated).
func (msg *MsgRegisterPQCKey) ProtoMessage()             {}
func (msg *MsgRegisterPQCKey) Reset()                    {}
func (msg *MsgRegisterPQCKey) String() string             { return "MsgRegisterPQCKey" }
