package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgCrossVMCall triggers a cross-VM contract call.
// This type will be promoted to a proper sdk.Msg (with proto.Message)
// once protobuf definitions are generated.
type MsgCrossVMCall struct {
	Sender         string    `json:"sender"`
	SourceVM       VMType    `json:"source_vm"`
	TargetVM       VMType    `json:"target_vm"`
	TargetContract string    `json:"target_contract"`
	Payload        []byte    `json:"payload"`
	Funds          sdk.Coins `json:"funds"`
}

func (m *MsgCrossVMCall) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidMessage, "invalid sender address: %s", err)
	}
	if m.SourceVM != VMTypeEVM && m.SourceVM != VMTypeCosmWasm {
		return errorsmod.Wrapf(ErrUnsupportedVM, "invalid source VM: %s", m.SourceVM)
	}
	if m.TargetVM != VMTypeEVM && m.TargetVM != VMTypeCosmWasm {
		return errorsmod.Wrapf(ErrUnsupportedVM, "invalid target VM: %s", m.TargetVM)
	}
	if m.SourceVM == m.TargetVM {
		return errorsmod.Wrap(ErrInvalidMessage, "source and target VM must differ")
	}
	if m.TargetContract == "" {
		return errorsmod.Wrap(ErrInvalidTarget, "target contract is required")
	}
	if len(m.Payload) == 0 {
		return errorsmod.Wrap(ErrInvalidMessage, "payload is required")
	}
	if !m.Funds.IsValid() {
		return errorsmod.Wrap(ErrInvalidMessage, "invalid funds")
	}
	return nil
}

// MsgProcessQueue triggers processing of the pending cross-VM message queue.
// This is typically called by the module's EndBlocker automatically,
// but can also be triggered manually by an authorized account.
type MsgProcessQueue struct {
	Authority string `json:"authority"`
}

func (m *MsgProcessQueue) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidMessage, "invalid authority address: %s", err)
	}
	return nil
}
