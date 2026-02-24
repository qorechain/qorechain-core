package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------------------------------------------------------------------------
// Message Types
// ---------------------------------------------------------------------------

// MsgDeployProgram deploys a BPF program to the SVM runtime.
type MsgDeployProgram struct {
	Sender   string `json:"sender"`
	Bytecode []byte `json:"bytecode"`
}

// ValidateBasic performs stateless validation of a program deployment message.
func (m *MsgDeployProgram) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	if len(m.Bytecode) == 0 {
		return errorsmod.Wrap(ErrInvalidBytecode, "bytecode cannot be empty")
	}
	return nil
}

func (m *MsgDeployProgram) Reset()         { *m = MsgDeployProgram{} }
func (m *MsgDeployProgram) String() string { return fmt.Sprintf("MsgDeployProgram{sender=%s}", m.Sender) }
func (m *MsgDeployProgram) ProtoMessage()  {}

// MsgExecuteProgram executes an instruction on a deployed SVM program.
type MsgExecuteProgram struct {
	Sender    string        `json:"sender"`
	ProgramID [32]byte      `json:"program_id"`
	Accounts  []AccountMeta `json:"accounts"`
	Data      []byte        `json:"data"`
}

// ValidateBasic performs stateless validation of a program execution message.
func (m *MsgExecuteProgram) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	var zeroAddr [32]byte
	if m.ProgramID == zeroAddr {
		return errorsmod.Wrap(ErrInvalidInstruction, "program ID cannot be zero")
	}
	return nil
}

func (m *MsgExecuteProgram) Reset()         { *m = MsgExecuteProgram{} }
func (m *MsgExecuteProgram) String() string { return fmt.Sprintf("MsgExecuteProgram{sender=%s}", m.Sender) }
func (m *MsgExecuteProgram) ProtoMessage()  {}

// MsgCreateAccount creates a new SVM data account with allocated space.
type MsgCreateAccount struct {
	Sender   string   `json:"sender"`
	Owner    [32]byte `json:"owner"`
	Space    uint64   `json:"space"`
	Lamports uint64   `json:"lamports"`
}

// ValidateBasic performs stateless validation of an account creation message.
func (m *MsgCreateAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	var zeroAddr [32]byte
	if m.Owner == zeroAddr {
		return errorsmod.Wrap(ErrInvalidAccountOwner, "owner cannot be zero")
	}
	return nil
}

func (m *MsgCreateAccount) Reset()         { *m = MsgCreateAccount{} }
func (m *MsgCreateAccount) String() string { return fmt.Sprintf("MsgCreateAccount{sender=%s}", m.Sender) }
func (m *MsgCreateAccount) ProtoMessage()  {}

// MsgRegisterSVMPQCKey registers a Dilithium-5 key for optional PQC upgrade
// on an SVM account.
type MsgRegisterSVMPQCKey struct {
	Sender    string   `json:"sender"`
	SVMAddr   [32]byte `json:"svm_addr"`
	PQCPubKey []byte   `json:"pqc_pub_key"`
}

// ValidateBasic performs stateless validation of a PQC key registration message.
func (m *MsgRegisterSVMPQCKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid sender: %s", err)
	}
	var zeroAddr [32]byte
	if m.SVMAddr == zeroAddr {
		return errorsmod.Wrap(ErrInvalidAddress, "SVM address cannot be zero")
	}
	if len(m.PQCPubKey) == 0 {
		return errorsmod.Wrap(ErrInvalidSignature, "PQC public key cannot be empty")
	}
	return nil
}

func (m *MsgRegisterSVMPQCKey) Reset()         { *m = MsgRegisterSVMPQCKey{} }
func (m *MsgRegisterSVMPQCKey) String() string { return fmt.Sprintf("MsgRegisterSVMPQCKey{sender=%s}", m.Sender) }
func (m *MsgRegisterSVMPQCKey) ProtoMessage()  {}

// ---------------------------------------------------------------------------
// Codec Registration
// ---------------------------------------------------------------------------

// RegisterInterfaces registers the SVM module's interface types.
func RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// TODO: Register sdk.Msg implementations once proto definitions are generated.
}

// RegisterLegacyAminoCodec registers the SVM module's types with the legacy amino codec.
func RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {
	// TODO: Register concrete types once proto definitions are generated.
}
