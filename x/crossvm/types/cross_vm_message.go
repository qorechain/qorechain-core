package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VMType identifies which virtual machine a contract runs on.
type VMType string

const (
	VMTypeEVM      VMType = "evm"
	VMTypeCosmWasm VMType = "cosmwasm"
	VMTypeSVM      VMType = "svm"

	// VMTypeNative is the origin of a call that did not come from a contract on
	// any VM: an account initiating one directly through a Cosmos transaction.
	// It is a SOURCE ONLY and is rejected as a target, because there is no
	// "native VM" to route a call into.
	//
	// It exists so the source lane can be stated truthfully. Before it, an
	// account submitting MsgCrossVMCall had to declare itself as one of the three
	// VMs, and nothing checked the claim, so the field recorded whatever the
	// sender preferred. A target that reads it to decide what to trust was
	// reading the caller's own assertion about itself.
	VMTypeNative VMType = "native"
)

// IsValidSourceVM reports whether v can be the origin of a cross-VM call.
func IsValidSourceVM(v VMType) bool {
	return v == VMTypeEVM || v == VMTypeCosmWasm || v == VMTypeSVM || v == VMTypeNative
}

// IsValidTargetVM reports whether a call can be routed into v. Narrower than the
// source set: a call has to land in a runtime that can execute it.
func IsValidTargetVM(v VMType) bool {
	return v == VMTypeEVM || v == VMTypeCosmWasm || v == VMTypeSVM
}

// MessageStatus tracks the lifecycle of a cross-VM message.
type MessageStatus string

const (
	StatusPending  MessageStatus = "pending"
	StatusExecuted MessageStatus = "executed"
	StatusFailed   MessageStatus = "failed"
	StatusTimedOut MessageStatus = "timed_out"
)

// CrossVMMessage represents a message passed between VMs.
type CrossVMMessage struct {
	ID             string        `json:"id"`
	SourceVM       VMType        `json:"source_vm"`
	TargetVM       VMType        `json:"target_vm"`
	Sender         string        `json:"sender"`
	SourceContract string        `json:"source_contract"`
	TargetContract string        `json:"target_contract"`
	Payload        []byte        `json:"payload"`
	Funds          sdk.Coins     `json:"funds"`
	Status         MessageStatus `json:"status"`
	CreatedHeight  int64         `json:"created_height"`
	ExecutedHeight int64         `json:"executed_height,omitempty"`
	Response       []byte        `json:"response,omitempty"`
	Error          string        `json:"error,omitempty"`
	// OnBehalf records that the call was sent on the sender's behalf, not by
	// the sender itself at top level (crossvm-on-behalf-evm-window): a call
	// into the EVM then runs under the sender's EVM authorisation window.
	// Set by the message server only; omitempty keeps older records
	// unchanged.
	OnBehalf bool `json:"on_behalf,omitempty"`
	// AsyncGas is the contract execution an async call into CosmWasm runs
	// on, in base gas units (zero: AsyncCallGas); PrepaidGas is what it paid
	// for it at submission, that times the congestion factor then (zero:
	// AsyncCallGas), from crossvm-async-gas-priced. omitempty keeps older
	// records unchanged.
	AsyncGas   uint64 `json:"async_gas,omitempty"`
	PrepaidGas uint64 `json:"prepaid_gas,omitempty"`
	// From crossvm-async-escrow: MaxFactor is the highest congestion factor
	// (millionths) the call runs at; Escrow is what the sender holds in the
	// queue's escrow account for its prepaid gas while the call is pending
	// (EscrowGasPrice per gas, in the fee denomination); EscrowFee is what
	// the finished call paid from it to validators (the rest went back to
	// the sender). A finished call holds no escrow.
	MaxFactor      uint64    `json:"max_factor,omitempty"`
	Escrow         sdk.Coins `json:"escrow,omitempty"`
	EscrowGasPrice string    `json:"escrow_gas_price,omitempty"`
	EscrowFee      sdk.Coins `json:"escrow_fee,omitempty"`
	// LoadUnits and LoadGas are what loading the target code costs (budget
	// units, and gas at the base price; zero for a pinned code), recorded at
	// submission of an escrowed call into CosmWasm (wasm-code-load-priced)
	// and included in its prepay and its admission.
	LoadUnits uint64 `json:"load_units,omitempty"`
	LoadGas   uint64 `json:"load_gas,omitempty"`
}

func (m CrossVMMessage) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}
	if !IsValidSourceVM(m.SourceVM) {
		return fmt.Errorf("invalid source VM: %s", m.SourceVM)
	}
	if !IsValidTargetVM(m.TargetVM) {
		return fmt.Errorf("invalid target VM: %s", m.TargetVM)
	}
	if m.SourceVM == m.TargetVM {
		return fmt.Errorf("source and target VM must be different")
	}
	// The sender is the identity the target VM executes the call as, so it has to
	// be a real account address in the canonical encoding. This was unchecked: a
	// message could be stored with an empty or malformed sender and only fail
	// later, inside the target lane, when the address was finally parsed.
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender %q: %w", m.Sender, err)
	}
	if m.TargetContract == "" {
		return fmt.Errorf("target contract cannot be empty")
	}
	if len(m.Payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
	}
	if !m.Escrow.IsValid() || !m.EscrowFee.IsValid() {
		return fmt.Errorf("invalid escrow")
	}
	if !m.Escrow.IsZero() && m.Status != StatusPending {
		return fmt.Errorf("a finished call cannot hold escrow")
	}
	return nil
}

// CrossVMResponse is the result of executing a cross-VM call.
type CrossVMResponse struct {
	MessageID string `json:"message_id"`
	Success   bool   `json:"success"`
	Data      []byte `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	GasUsed   uint64 `json:"gas_used"`
}

// MarshalCrossVMMessage serializes a CrossVMMessage to JSON bytes.
func MarshalCrossVMMessage(msg *CrossVMMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// UnmarshalCrossVMMessage deserializes JSON bytes into a CrossVMMessage.
func UnmarshalCrossVMMessage(bz []byte) (*CrossVMMessage, error) {
	var msg CrossVMMessage
	if err := json.Unmarshal(bz, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
