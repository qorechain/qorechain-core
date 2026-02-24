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
)

// MessageStatus tracks the lifecycle of a cross-VM message.
type MessageStatus string

const (
	StatusPending   MessageStatus = "pending"
	StatusExecuted  MessageStatus = "executed"
	StatusFailed    MessageStatus = "failed"
	StatusTimedOut  MessageStatus = "timed_out"
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
	if m.SourceVM != VMTypeEVM && m.SourceVM != VMTypeCosmWasm && m.SourceVM != VMTypeSVM {
		return fmt.Errorf("invalid source VM: %s", m.SourceVM)
	}
	if m.TargetVM != VMTypeEVM && m.TargetVM != VMTypeCosmWasm && m.TargetVM != VMTypeSVM {
		return fmt.Errorf("invalid target VM: %s", m.TargetVM)
	}
	if m.SourceVM == m.TargetVM {
		return fmt.Errorf("source and target VM must be different")
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
