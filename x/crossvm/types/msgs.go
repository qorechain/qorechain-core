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
	if m.AsyncGasLimit != 0 {
		if !m.Async || m.TargetVM != VMTypeCosmWasm {
			return errorsmod.Wrap(ErrInvalidMessage, "async_gas_limit is for async calls into CosmWasm only")
		}
		if m.AsyncGasLimit < MinAsyncGasLimit || m.AsyncGasLimit > MaxAsyncGasLimit {
			return errorsmod.Wrapf(ErrInvalidMessage, "async_gas_limit %d is outside %d..%d", m.AsyncGasLimit, MinAsyncGasLimit, MaxAsyncGasLimit)
		}
	}
	if m.AsyncMaxFactor != 0 {
		if !m.Async || m.TargetVM != VMTypeCosmWasm {
			return errorsmod.Wrap(ErrInvalidMessage, "async_max_factor is for async calls into CosmWasm only")
		}
		if max := uint64(MaxWasmCongestionMaxFactor) * CongestionFactorOne; m.AsyncMaxFactor < CongestionFactorOne || m.AsyncMaxFactor > max {
			return errorsmod.Wrapf(ErrInvalidMessage, "async_max_factor %d is outside %d..%d (millionths)", m.AsyncMaxFactor, CongestionFactorOne, max)
		}
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

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidMessage, "invalid authority address: %s", err)
	}
	return m.Params().ValidateUpdate()
}

// Params returns the parameter set the message installs.
func (m *MsgUpdateParams) Params() Params {
	return Params{
		MaxMessageSize:          m.MaxMessageSize,
		MaxQueueSize:            m.MaxQueueSize,
		QueueTimeoutBlocks:      m.QueueTimeoutBlocks,
		Enabled:                 m.Enabled,
		BridgeAllowedTargets:    m.BridgeAllowedTargets,
		BridgeTargetPins:        m.BridgeTargetPins,
		MaxPendingPerSender:     m.MaxPendingPerSender,
		MaxCosmosTxGas:          m.MaxCosmosTxGas,
		RecordRetentionBlocks:   m.RecordRetentionBlocks,
		MaxBlockWasmGas:         m.MaxBlockWasmGas,
		MaxTxWasmShareBps:       m.MaxTxWasmShareBps,
		MaxBlockLevelWasmGas:    m.MaxBlockLevelWasmGas,
		QueueWasmReserveBps:     m.QueueWasmReserveBps,
		WasmCongestionMaxFactor: m.WasmCongestionMaxFactor,
		WasmCongestionTargetBps: m.WasmCongestionTargetBps,
		MaxBlockUploadBps:       m.MaxBlockUploadBps,
	}
}
