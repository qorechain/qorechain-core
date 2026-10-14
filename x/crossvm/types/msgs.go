package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgCrossVMCall and MsgProcessQueue are generated from
// proto/qorechain/crossvm/v1/tx.proto (see tx.pb.go). The ValidateBasic
// methods below are attached to those generated types.

func (m *MsgCrossVMCall) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(ErrInvalidMessage, "invalid sender address: %s", err)
	}
	if m.SourceVM != VMTypeEVM && m.SourceVM != VMTypeCosmWasm && m.SourceVM != VMTypeSVM {
		return errorsmod.Wrapf(ErrUnsupportedVM, "invalid source VM: %s", m.SourceVM)
	}
	if m.TargetVM != VMTypeEVM && m.TargetVM != VMTypeCosmWasm && m.TargetVM != VMTypeSVM {
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

// PlaceholderSourceVM returns a source lane that satisfies the stateless
// MsgCrossVMCall.ValidateBasic for a call into target.
//
// The chain ignores source_vm: the keeper derives the caller's lane from state
// (SourceLaneOf). ValidateBasic still requires a VM name that differs from the
// target, and it cannot be relaxed without changing which transactions every
// earlier block accepted, so a client that has nothing to declare fills in this
// placeholder. The value carries no meaning and is never read by the chain.
func PlaceholderSourceVM(target VMType) VMType {
	for _, vm := range []VMType{VMTypeEVM, VMTypeCosmWasm, VMTypeSVM} {
		if vm != target {
			return vm
		}
	}
	return VMTypeEVM
}

func (m *MsgProcessQueue) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidMessage, "invalid authority address: %s", err)
	}
	return nil
}
