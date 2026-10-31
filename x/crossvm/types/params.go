package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultMaxMessageSize     uint64 = 65536 // 64KB
	DefaultMaxQueueSize       uint32 = 1000
	DefaultQueueTimeoutBlocks int64  = 100 // messages expire after 100 blocks
	DefaultEnabled            bool   = true

	// Ceilings MsgUpdateParams holds governance to (ValidateUpdate). Genesis
	// and stored params are not held to them, so no existing chain's state
	// changes meaning.
	MaxUpdateMessageSize       uint64 = 1 << 20 // 1 MiB
	MaxUpdateQueueSize         uint32 = 10_000
	MaxUpdateQueueTimeout      int64  = 100_000
	MaxBridgeAllowedTargets           = 64
	MaxBridgeAllowedTargetSize        = 256

	// AsyncCallGas is the execution budget an async MsgCrossVMCall prepays from
	// the crossvm-lane-bounds switch: it is charged to the submitting
	// transaction, and the end-of-block queue runs the call on a meter of
	// exactly this size. A queued call that needs more fails; one that needs
	// more than this should be made synchronously, where the caller's own gas
	// limit applies.
	AsyncCallGas uint64 = 1_000_000
	// MaxAsyncCallsPerBlock bounds how many queued calls one EndBlock executes
	// from the same switch, so the end-of-block work is at most
	// MaxAsyncCallsPerBlock * AsyncCallGas whatever the queue holds. The rest
	// stay pending for the next block (and still expire at their timeout).
	MaxAsyncCallsPerBlock = 50
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

// ValidateUpdate is Validate plus the bounds a governance update must respect:
// numeric ceilings, a bounded allowlist with no duplicates, and allowlist
// entries the bridge can actually reach (CosmWasm targets, by account
// address). The bridge precompile only calls CosmWasm, so any other entry
// would be dead weight that looks like an open door.
func (p Params) ValidateUpdate() error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.MaxMessageSize > MaxUpdateMessageSize {
		return fmt.Errorf("max message size %d exceeds %d", p.MaxMessageSize, MaxUpdateMessageSize)
	}
	if p.MaxQueueSize > MaxUpdateQueueSize {
		return fmt.Errorf("max queue size %d exceeds %d", p.MaxQueueSize, MaxUpdateQueueSize)
	}
	if p.QueueTimeoutBlocks > MaxUpdateQueueTimeout {
		return fmt.Errorf("queue timeout %d exceeds %d blocks", p.QueueTimeoutBlocks, MaxUpdateQueueTimeout)
	}
	if len(p.BridgeAllowedTargets) > MaxBridgeAllowedTargets {
		return fmt.Errorf("bridge allowlist has %d entries, max %d", len(p.BridgeAllowedTargets), MaxBridgeAllowedTargets)
	}
	seen := make(map[string]bool, len(p.BridgeAllowedTargets))
	for _, entry := range p.BridgeAllowedTargets {
		if len(entry) > MaxBridgeAllowedTargetSize {
			return fmt.Errorf("bridge allowed target %q is longer than %d bytes", entry, MaxBridgeAllowedTargetSize)
		}
		if seen[entry] {
			return fmt.Errorf("bridge allowed target %q is listed twice", entry)
		}
		seen[entry] = true
		vm, contract, _ := strings.Cut(entry, ":")
		if VMType(vm) != VMTypeCosmWasm {
			return fmt.Errorf("bridge allowed target %q: the bridge only calls %s contracts", entry, VMTypeCosmWasm)
		}
		if _, err := sdk.AccAddressFromBech32(contract); err != nil {
			return fmt.Errorf("bridge allowed target %q: invalid contract address: %w", entry, err)
		}
	}
	return nil
}
