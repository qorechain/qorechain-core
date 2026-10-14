package types

import (
	"fmt"
	"strings"
)

const (
	DefaultMaxMessageSize     uint64 = 65536 // 64KB
	DefaultMaxQueueSize       uint32 = 1000
	DefaultQueueTimeoutBlocks int64  = 100 // messages expire after 100 blocks
	DefaultEnabled            bool   = true
)

// Params defines the parameters for the crossvm module.
type Params struct {
	MaxMessageSize     uint64 `json:"max_message_size"`
	MaxQueueSize       uint32 `json:"max_queue_size"`
	QueueTimeoutBlocks int64  `json:"queue_timeout_blocks"`
	Enabled            bool   `json:"enabled"`

	// BridgeAllowedTargets lists the targets the EVM CrossVM bridge precompile
	// may call, each as "<vm>:<contract>" (for example "cosmwasm:qor1...").
	// From the evm-precompile-hardening switch the bridge refuses every other
	// target, so an empty list keeps the bridge shut even if the precompile is
	// ever activated. omitempty keeps the stored and exported params of every
	// existing chain byte-identical.
	BridgeAllowedTargets []string `json:"bridge_allowed_targets,omitempty"`
}

func DefaultParams() Params {
	return Params{
		MaxMessageSize:     DefaultMaxMessageSize,
		MaxQueueSize:       DefaultMaxQueueSize,
		QueueTimeoutBlocks: DefaultQueueTimeoutBlocks,
		Enabled:            DefaultEnabled,
	}
}

func (p Params) Validate() error {
	if p.MaxMessageSize == 0 {
		return fmt.Errorf("max message size must be positive")
	}
	if p.MaxQueueSize == 0 {
		return fmt.Errorf("max queue size must be positive")
	}
	if p.QueueTimeoutBlocks <= 0 {
		return fmt.Errorf("queue timeout blocks must be positive")
	}
	for _, entry := range p.BridgeAllowedTargets {
		vm, contract, ok := strings.Cut(entry, ":")
		if !ok || contract == "" || !IsValidTargetVM(VMType(vm)) {
			return fmt.Errorf("bridge allowed target %q must be <vm>:<contract>", entry)
		}
	}
	return nil
}

// BridgeTargetAllowed reports whether the EVM bridge precompile may call
// contract on vm.
func (p Params) BridgeTargetAllowed(vm VMType, contract string) bool {
	if contract == "" {
		return false
	}
	want := string(vm) + ":" + contract
	for _, entry := range p.BridgeAllowedTargets {
		if entry == want {
			return true
		}
	}
	return false
}
