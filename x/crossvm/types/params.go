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

	// DefaultMaxPendingPerSender is how many async calls one sender may have
	// queued at a time when params.MaxPendingPerSender is zero, from the
	// crossvm-queue-sender-cap switch. A tenth of the default queue, so at
	// least ten distinct senders are needed to fill it; anyone who needs more
	// in flight can make the calls synchronously.
	DefaultMaxPendingPerSender uint32 = 100
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

	// BridgeTargetPins binds allowlist entries to the CosmWasm code their
	// contract must run (crossvm-bridge-code-pin). omitempty keeps the stored
	// params of every chain that never set one byte-identical.
	BridgeTargetPins []BridgeTargetPin `json:"bridge_target_pins,omitempty"`

	// MaxPendingPerSender bounds the async calls one sender may have queued at
	// a time (crossvm-queue-sender-cap); zero means
	// DefaultMaxPendingPerSender. omitempty as above.
	MaxPendingPerSender uint32 `json:"max_pending_per_sender,omitempty"`
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
	return p.validatePins()
}

// validatePins holds each pin to an entry of the allowlist, once, with a
// code ID. Params that carry no pin (every chain before the
// crossvm-bridge-code-pin switch) pass unchanged.
func (p Params) validatePins() error {
	if len(p.BridgeTargetPins) == 0 {
		return nil
	}
	listed := make(map[string]bool, len(p.BridgeAllowedTargets))
	for _, entry := range p.BridgeAllowedTargets {
		listed[entry] = true
	}
	pinned := make(map[string]bool, len(p.BridgeTargetPins))
	for _, pin := range p.BridgeTargetPins {
		if !listed[pin.Target] {
			return fmt.Errorf("bridge target pin %q names no allowlist entry", pin.Target)
		}
		if pinned[pin.Target] {
			return fmt.Errorf("bridge target %q is pinned twice", pin.Target)
		}
		pinned[pin.Target] = true
		if pin.CodeID == 0 {
			return fmt.Errorf("bridge target pin %q has no code ID", pin.Target)
		}
		if pin.HistoryLen == 0 {
			return fmt.Errorf("bridge target pin %q has no code history length", pin.Target)
		}
	}
	return nil
}

// BridgeTargetCodePin returns the pin of the allowlist entry, if any.
func (p Params) BridgeTargetCodePin(entry string) (BridgeTargetPin, bool) {
	for _, pin := range p.BridgeTargetPins {
		if pin.Target == entry {
			return pin, true
		}
	}
	return BridgeTargetPin{}, false
}

// PendingPerSenderLimit is the per-sender bound on queued async calls in
// force: MaxPendingPerSender, or DefaultMaxPendingPerSender when unset.
func (p Params) PendingPerSenderLimit() uint32 {
	if p.MaxPendingPerSender == 0 {
		return DefaultMaxPendingPerSender
	}
	return p.MaxPendingPerSender
}

// BridgeContractState is what the bridge code pin needs to know about the
// contract an allowlist entry names, read from the CosmWasm keeper.
// HistoryLen is the number of entries of the contract's code history (wasmd
// ContractCodeHistory): instantiation and every migration append one.
type BridgeContractState struct {
	Found      bool
	CodeID     uint64
	Admin      string
	HistoryLen uint64
}

// BridgeContractLookup reads a contract's state by its bech32 address.
type BridgeContractLookup func(contract string) BridgeContractState

// adminIsGovOrNone reports whether nobody but governance can migrate a
// contract with this admin.
func adminIsGovOrNone(admin, govAuthority string) bool {
	return admin == "" || admin == govAuthority
}

// pinMatches reports why st does not match pin, or nil when it does: the
// contract must exist, run the pinned code, and have exactly the code history
// the pin was taken at. The history check is what binds the pin to state as
// well as code: an admin who migrates the contract to other code (whose
// migrate entry point can rewrite its storage) and back again leaves the code
// ID as pinned but the history two entries longer.
func pinMatches(entry string, pin BridgeTargetPin, st BridgeContractState) error {
	if !st.Found {
		return fmt.Errorf("target %s is pinned to code %d but names no contract", entry, pin.CodeID)
	}
	if st.CodeID != pin.CodeID {
		return fmt.Errorf("target %s is pinned to code %d but its contract now runs code %d", entry, pin.CodeID, st.CodeID)
	}
	if st.HistoryLen != pin.HistoryLen {
		return fmt.Errorf("target %s is pinned at code history length %d but its contract has been migrated since (history length %d)", entry, pin.HistoryLen, st.HistoryLen)
	}
	return nil
}

// CheckBridgeTargetCode is the call-time half of crossvm-bridge-code-pin, for
// an entry that BridgeTargetAllowed already matched. A pinned entry is called
// only while its contract runs the pinned code with the pinned code history.
// An unpinned entry is called only while its contract exists and nobody but
// governance can migrate it (no admin, or governance as admin).
func (p Params) CheckBridgeTargetCode(vm VMType, contract string, st BridgeContractState, govAuthority string) error {
	entry := string(vm) + ":" + contract
	if pin, ok := p.BridgeTargetCodePin(entry); ok {
		return pinMatches(entry, pin, st)
	}
	if !st.Found {
		return fmt.Errorf("target %s names no contract", entry)
	}
	if !adminIsGovOrNone(st.Admin, govAuthority) {
		return fmt.Errorf("target %s is not pinned to a code and its admin %s is not governance", entry, st.Admin)
	}
	return nil
}

// ValidateBridgePins is the parameter-update half of crossvm-bridge-code-pin:
// every allowlist entry names an existing contract, a pin names the code the
// contract runs now and its current code history length, and an entry whose
// contract an account other than governance could migrate is pinned. A
// governance proposal that migrates a pinned contract updates the pin after
// the migration message.
func (p Params) ValidateBridgePins(lookup BridgeContractLookup, govAuthority string) error {
	for _, entry := range p.BridgeAllowedTargets {
		_, contract, _ := strings.Cut(entry, ":")
		st := lookup(contract)
		if !st.Found {
			return fmt.Errorf("bridge allowed target %q names no contract", entry)
		}
		if pin, ok := p.BridgeTargetCodePin(entry); ok {
			if err := pinMatches(entry, pin, st); err != nil {
				return fmt.Errorf("bridge allowed target %q: %s", entry, err.Error())
			}
			continue
		}
		if !adminIsGovOrNone(st.Admin, govAuthority) {
			return fmt.Errorf("bridge allowed target %q has admin %s, which is not governance: pin its code and code history length (bridge_target_pins) to allowlist it", entry, st.Admin)
		}
	}
	return nil
}

// PinBridgeTargets is the v3.2.8 release step that carries the existing
// allowlist over to crossvm-bridge-code-pin. Every entry whose contract exists
// is pinned to the code it runs now and its current code history length, so
// every target that answered before still answers. An entry that names no
// contract is removed (with any pin it had): left unpinned, it would be
// callable as soon as someone instantiated a contract at that address, with
// whatever initial state they chose. It returns the new params, whether
// anything changed and the removed entries.
func (p Params) PinBridgeTargets(lookup BridgeContractLookup) (Params, bool, []string) {
	var (
		targets []string
		pins    []BridgeTargetPin
		removed []string
	)
	for _, entry := range p.BridgeAllowedTargets {
		_, contract, _ := strings.Cut(entry, ":")
		st := lookup(contract)
		if !st.Found {
			removed = append(removed, entry)
			continue
		}
		targets = append(targets, entry)
		if pin, ok := p.BridgeTargetCodePin(entry); ok {
			pins = append(pins, pin)
			continue
		}
		pins = append(pins, BridgeTargetPin{Target: entry, CodeID: st.CodeID, HistoryLen: st.HistoryLen})
	}
	if len(removed) == 0 && len(pins) == len(p.BridgeTargetPins) {
		return p, false, nil
	}
	p.BridgeAllowedTargets = targets
	p.BridgeTargetPins = pins
	return p, true, removed
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
	if p.MaxPendingPerSender > p.MaxQueueSize {
		return fmt.Errorf("max pending per sender %d exceeds the queue size %d", p.MaxPendingPerSender, p.MaxQueueSize)
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
