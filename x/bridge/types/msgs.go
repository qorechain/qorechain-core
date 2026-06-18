package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The MsgBridgeDeposit, MsgBridgeWithdraw, MsgRegisterBridgeValidator and
// MsgBridgeAttestation structs are generated from
// proto/qorechain/bridge/v1/tx.proto (see tx.pb.go). The ValidateBasic,
// GetSigners and GetSignBytes methods below are attached to those generated
// types.

// Message type constants.
const (
	TypeMsgBridgeDeposit           = "bridge_deposit"
	TypeMsgBridgeWithdraw          = "bridge_withdraw"
	TypeMsgRegisterBridgeValidator = "register_bridge_validator"
	TypeMsgBridgeAttestation       = "bridge_attestation"
)

func (msg MsgBridgeDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if msg.SourceChain == "" {
		return ErrChainNotSupported.Wrap("source chain cannot be empty")
	}
	if msg.Asset == "" {
		return ErrAssetNotSupported.Wrap("asset cannot be empty")
	}
	if msg.Amount == "" {
		return ErrInvalidAmount.Wrap("amount cannot be empty")
	}
	if _, err := ParseAmount(msg.Amount); err != nil {
		return err
	}
	if msg.SourceTxHash == "" {
		return ErrInvalidAttestation.Wrap("source_tx_hash cannot be empty")
	}
	if len(msg.SourceTxHash) > 128 {
		return ErrInvalidAttestation.Wrapf("source_tx_hash too long: %d, max 128", len(msg.SourceTxHash))
	}
	return nil
}

func (msg MsgBridgeDeposit) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgBridgeWithdraw) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}
	if msg.DestinationChain == "" {
		return ErrChainNotSupported.Wrap("destination chain cannot be empty")
	}
	if msg.DestinationAddress == "" {
		return ErrInvalidDestination.Wrap("destination address cannot be empty")
	}
	if msg.Asset == "" {
		return ErrAssetNotSupported.Wrap("asset cannot be empty")
	}
	if msg.Amount == "" {
		return ErrInvalidAmount.Wrap("amount cannot be empty")
	}
	if _, err := ParseAmount(msg.Amount); err != nil {
		return err
	}
	return nil
}

func (msg MsgBridgeWithdraw) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgRegisterBridgeValidator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ValidatorAddress); err != nil {
		return err
	}
	if len(msg.PQCPubkey) != 2592 {
		return ErrInvalidPQCSignature.Wrapf("expected 2592-byte Dilithium-5 pubkey, got %d", len(msg.PQCPubkey))
	}
	if len(msg.SupportedChains) == 0 {
		return ErrChainNotSupported.Wrap("must support at least one chain")
	}
	if len(msg.SupportedChains) > 25 {
		return ErrInvalidAttestation.Wrapf("too many supported chains: %d, max 25", len(msg.SupportedChains))
	}
	return nil
}

func (msg MsgRegisterBridgeValidator) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgBridgeAttestation) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Validator); err != nil {
		return err
	}
	if msg.Chain == "" {
		return ErrChainNotSupported.Wrap("chain cannot be empty")
	}
	if msg.OperationID == "" {
		return ErrOperationNotFound.Wrap("operation ID cannot be empty")
	}
	if len(msg.PQCSignature) != 4627 {
		return ErrInvalidPQCSignature.Wrapf("expected 4627-byte Dilithium-5 signature, got %d", len(msg.PQCSignature))
	}
	return nil
}

func (msg MsgBridgeAttestation) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Validator)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

// GetSignBytes returns the attestation data that was signed.
func (msg MsgBridgeAttestation) GetSignBytes() []byte {
	// Deterministic byte representation for PQC signature verification
	return []byte(fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		msg.Chain, msg.EventType, msg.OperationID,
		msg.TxHash, msg.Amount.String(), msg.Asset))
}
