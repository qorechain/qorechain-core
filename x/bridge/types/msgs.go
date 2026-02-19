package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message type constants.
const (
	TypeMsgBridgeDeposit           = "bridge_deposit"
	TypeMsgBridgeWithdraw          = "bridge_withdraw"
	TypeMsgRegisterBridgeValidator = "register_bridge_validator"
	TypeMsgBridgeAttestation       = "bridge_attestation"
)

// MsgBridgeDeposit initiates a deposit from an external chain.
type MsgBridgeDeposit struct {
	Sender              string `json:"sender"`
	SourceChain         string `json:"source_chain"`
	SourceTxHash        string `json:"source_tx_hash"`
	Asset               string `json:"asset"`
	Amount              string `json:"amount"`
	BridgeValidatorSigs []byte `json:"bridge_validator_sigs"`
	PQCCommitment       []byte `json:"pqc_commitment"`
}

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
	return nil
}

func (msg MsgBridgeDeposit) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

// MsgBridgeWithdraw initiates a withdrawal to an external chain.
type MsgBridgeWithdraw struct {
	Sender             string `json:"sender"`
	DestinationChain   string `json:"destination_chain"`
	DestinationAddress string `json:"destination_address"`
	Asset              string `json:"asset"`
	Amount             string `json:"amount"`
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
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

// MsgRegisterBridgeValidator registers a validator for bridge operations.
type MsgRegisterBridgeValidator struct {
	ValidatorAddress string   `json:"validator_address"`
	PQCPubkey        []byte   `json:"pqc_pubkey"`
	SupportedChains  []string `json:"supported_chains"`
}

func (msg MsgRegisterBridgeValidator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ValidatorAddress); err != nil {
		return err
	}
	if len(msg.PQCPubkey) == 0 {
		return ErrInvalidPQCSignature.Wrap("PQC pubkey cannot be empty")
	}
	if len(msg.SupportedChains) == 0 {
		return ErrChainNotSupported.Wrap("must support at least one chain")
	}
	return nil
}

func (msg MsgRegisterBridgeValidator) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.ValidatorAddress)
	return []sdk.AccAddress{addr}
}

// MsgBridgeAttestation submits a validator attestation for a bridge event.
type MsgBridgeAttestation struct {
	Validator    string `json:"validator"`
	Chain        string `json:"chain"`
	EventType    string `json:"event_type"` // "deposit" | "withdrawal_complete"
	OperationID  string `json:"operation_id"`
	TxHash       string `json:"tx_hash"`
	Proof        []byte `json:"proof"`
	PQCSignature []byte `json:"pqc_signature"`
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
	if len(msg.PQCSignature) == 0 {
		return ErrInvalidPQCSignature.Wrap("PQC signature cannot be empty")
	}
	return nil
}

func (msg MsgBridgeAttestation) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Validator)
	return []sdk.AccAddress{addr}
}

// GetSignBytes returns the attestation data that was signed.
func (msg MsgBridgeAttestation) GetSignBytes() []byte {
	// Deterministic byte representation for PQC signature verification
	data := []byte(msg.Chain + "|" + msg.EventType + "|" + msg.OperationID + "|" + msg.TxHash)
	return data
}
