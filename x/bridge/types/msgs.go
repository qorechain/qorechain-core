package types

import (
	"encoding/binary"

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
// SignBytesForChain is the payload a bridge validator's ML-DSA key signs for an
// attestation.
//
// WHAT WAS WRONG WITH THE PREVIOUS ENCODING. It was a pipe-joined format string over
// six fields, with no length prefixes and no chain-id. None of the fields is
// forbidden from containing a pipe, so field boundaries could be moved: two
// distinct operations could serialise to identical bytes, and one valid
// attestation signature would then verify for the other. It also bound no chain, so
// an attestation signed for the testnet verified on mainnet if the same validator
// key was registered on both.
//
// Every field is now length-prefixed with BE64, the same framing every other signed
// payload in this codebase uses (x/svm/types/auth_sign.go, x/abstractaccount sign
// bytes), a domain tag prevents the bytes from being valid as anything else the key
// signs, and the chain-id is bound first. Distinct inputs always hash differently.
//
//	"qorechain-bridge-attestation-v2" ‖ BE64(len(chainID)) ‖ chainID ‖
//	  BE64(len(chain)) ‖ chain ‖ BE64(len(eventType)) ‖ eventType ‖
//	  BE64(len(operationID)) ‖ operationID ‖ BE64(len(txHash)) ‖ txHash ‖
//	  BE64(len(amount)) ‖ amount ‖ BE64(len(asset)) ‖ asset
//
// Bridge validators sign this with their registered key; the chain re-derives it
// from the stored operation, so an attestation that does not match the operation
// it claims to attest fails verification. The signed format changed, so this rides a
// coordinated upgrade: attestations produced under the old format are rejected by
// nodes running this, and validators must update their signer at the same height.
func (msg MsgBridgeAttestation) SignBytesForChain(chainID string) []byte {
	fields := []string{
		chainID, msg.Chain, msg.EventType, msg.OperationID,
		msg.TxHash, msg.Amount.String(), msg.Asset,
	}
	out := []byte("qorechain-bridge-attestation-v2")
	for _, f := range fields {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(f)))
		out = append(out, n[:]...)
		out = append(out, f...)
	}
	return out
}
