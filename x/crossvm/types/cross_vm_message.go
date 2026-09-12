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
