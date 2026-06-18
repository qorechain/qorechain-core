package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic implementations for the proto-generated SVM messages.

func (m *MsgDeployProgram) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	if len(m.Bytecode) == 0 {
		return errorsmod.Wrap(ErrInvalidBytecode, "bytecode cannot be empty")
	}
	return nil
}

func (m *MsgExecuteProgram) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	if m.ProgramID == (Bytes32{}) {
		return errorsmod.Wrap(ErrInvalidInstruction, "program ID cannot be zero")
	}
	return nil
}

func (m *MsgCreateAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	if m.Owner == (Bytes32{}) {
		return errorsmod.Wrap(ErrInvalidAccountOwner, "owner cannot be zero")
	}
	return nil
}

func (m *MsgRegisterSVMPQCKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	if m.SVMAddr == (Bytes32{}) {
		return errorsmod.Wrap(ErrInvalidAddress, "SVM address cannot be zero")
	}
	if len(m.PQCPubKey) == 0 {
		return errorsmod.Wrap(ErrInvalidSignature, "PQC public key cannot be empty")
	}
	return nil
}
