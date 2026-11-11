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

	// DefaultMaxCosmosTxGas is the gas limit one Cosmos-lane transaction may
	// declare when params.MaxCosmosTxGas is zero, from the cosmos-tx-gas-cap
	// switch. MinMaxCosmosTxGas is the lowest value governance may set, so
	// that no update can leave the chain unable to carry an ordinary
	// transaction (a vote, a parameter change).
	DefaultMaxCosmosTxGas uint64 = 50_000_000
	MinMaxCosmosTxGas     uint64 = 10_000_000

	// DefaultRecordRetentionBlocks is how long the record of a finished
	// queued call is kept when params.RecordRetentionBlocks is zero, from the
	// crossvm-record-prune switch (about 2.3 days at 2 s blocks).
	// MaxRecordPrunePerBlock bounds how many records one EndBlock deletes.
	DefaultRecordRetentionBlocks uint64 = 100_000
	MaxRecordPrunePerBlock              = 200

	// DefaultMaxBlockWasmGas is the CosmWasm execution budget of one block
	// (VM gas in SDK gas units) when params.MaxBlockWasmGas is zero, from the
	// wasm-block-gas-budget switch. The slowest operator class the VM admits
	// (memory.grow) costs 265 ns per unit at the calibrated price on an
	// Intel c6i and 257 on an AMD m6a (validator-class hardware), so
	// 3,000,000 bounds a worst-case block's wasm execution to about 0.85 s,
	// while an ordinary token transfer uses about 4,600 of it (several
	// hundred per block). MinMaxBlockWasmGas is the lowest value governance
	// may set. Changing the default is a one-line change here.
	DefaultMaxBlockWasmGas uint64 = 3_000_000
	MinMaxBlockWasmGas     uint64 = 1_000_000
	// MaxMaxBlockWasmGas, MaxMaxCosmosTxGas and MaxMaxBlockLevelWasmGas are
	// the highest values governance may set: well above any useful value
	// (50M of block wasm is about 13 s of worst-case execution on validator
	// hardware), and low enough that converting them to VM gas cannot
	// overflow at any multiplier wasmd uses (1e9 x 140,000 is far below
	// 2^64).
	MaxMaxBlockWasmGas      uint64 = 50_000_000
	MaxMaxCosmosTxGas       uint64 = 1_000_000_000
	MaxMaxBlockLevelWasmGas uint64 = 100_000_000

	// DefaultMaxTxWasmShareBps is the share of the block wasm budget one
	// transaction may use when params.MaxTxWasmShareBps is zero (half), so
	// one transaction cannot spend the whole budget and close the lane for
	// the rest of the block.
	DefaultMaxTxWasmShareBps uint32 = 5_000

	// DefaultMaxBlockLevelWasmGas bounds one contract call made outside a
	// transaction (a governance proposal executed at the end of the block),
	// which otherwise runs on the block's infinite gas meter, when
	// params.MaxBlockLevelWasmGas is zero: 10,000,000 is about 2.7 s of the
	// slowest admitted operator on validator hardware, and far above what an
	// instantiate, migrate or execute a proposal makes needs.
	// MinMaxBlockLevelWasmGas is the lowest value governance may set.
	DefaultMaxBlockLevelWasmGas uint64 = 10_000_000

	// DefaultQueueWasmReserveBps is the share of the block wasm budget
	// reserved for the end-of-block async queue when params.QueueWasmReserveBps
	// is zero: transactions cannot spend it, so queued CosmWasm calls are
	// never starved out by transactions.
	DefaultQueueWasmReserveBps uint32 = 2_500

	// MinQueueAdmission is the least gas a queued CosmWasm call is admitted
	// on: below it the call stays pending for a later block rather than run
	// on a meter too small to get through a contract's setup (which would
	// fail it, and every call after it, for nothing).
	MinQueueAdmission uint64 = AsyncCallGas / 4

	// MinAsyncGasLimit and MaxAsyncGasLimit bound the async_gas_limit a
	// sender may choose for an async call into CosmWasm
	// (crossvm-async-gas-priced). MaxAsyncPrepaid bounds what one async call
	// may prepay (its limit times the congestion factor): above it the call
	// is refused at submission (retry with a lower limit, or later) rather
	// than run a meter whose storage work the end-of-block queue could not
	// bound. MaxAsyncQueueGasPerBlock bounds the SDK gas all queued calls of
	// one block may run on together (the bound lane-bounds gave the queue).
	MinAsyncGasLimit         uint64 = MinQueueAdmission
	MaxAsyncGasLimit         uint64 = 10_000_000
	MaxAsyncPrepaid          uint64 = 50_000_000
	MaxAsyncQueueGasPerBlock uint64 = MaxAsyncCallsPerBlock * AsyncCallGas

	// Congestion pricing (wasm-congestion-pricing): a factor, kept in state,
	// by which one unit of contract execution costs more than the base
	// price. After every block it rises by 1/8 when the block used more than
	// the target share of its wasm budget and falls by 1/9 (back by the same
	// step) when it used less, between 1 and the maximum, so sustained
	// filling becomes exponentially expensive while ordinary use stays at
	// the base price. CongestionFactorOne is the factor 1 in millionths.
	DefaultWasmCongestionMaxFactor uint32 = 1_000
	DefaultWasmCongestionTargetBps uint32 = 5_000
	CongestionFactorOne            uint64 = 1_000_000

	// CongestionHoldBlocks is how many blocks after a block whose
	// transaction pool was exhausted the factor does not fall (hysteresis):
	// alternating closed and idle blocks cannot keep it near 1. A block that
	// exhausts the pool raises the factor by half instead of by 1/8.
	CongestionHoldBlocks    int64  = 10
	MinMaxBlockLevelWasmGas uint64 = 1_000_000
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

	// MaxCosmosTxGas bounds the gas limit of one Cosmos-lane transaction
	// (cosmos-tx-gas-cap); zero means DefaultMaxCosmosTxGas.
	MaxCosmosTxGas uint64 `json:"max_cosmos_tx_gas,omitempty"`

	// RecordRetentionBlocks is how long finished queued call records are kept
	// (crossvm-record-prune); zero means DefaultRecordRetentionBlocks.
	RecordRetentionBlocks uint64 `json:"record_retention_blocks,omitempty"`

	// MaxBlockWasmGas is the per-block CosmWasm execution budget
	// (wasm-block-gas-budget); zero means DefaultMaxBlockWasmGas.
	MaxBlockWasmGas uint64 `json:"max_block_wasm_gas,omitempty"`

	// MaxTxWasmShareBps is the share of the block wasm budget one
	// transaction may use, in basis points; zero means
	// DefaultMaxTxWasmShareBps.
	MaxTxWasmShareBps uint32 `json:"max_tx_wasm_share_bps,omitempty"`

	// MaxBlockLevelWasmGas bounds one contract call made outside a
	// transaction; zero means DefaultMaxBlockLevelWasmGas.
	MaxBlockLevelWasmGas uint64 `json:"max_block_level_wasm_gas,omitempty"`

	// QueueWasmReserveBps is the queue's reserved share of the block wasm
	// budget, in basis points; zero means DefaultQueueWasmReserveBps.
	QueueWasmReserveBps uint32 `json:"queue_wasm_reserve_bps,omitempty"`

	// WasmCongestionMaxFactor bounds the congestion factor; zero means
	// DefaultWasmCongestionMaxFactor.
	WasmCongestionMaxFactor uint32 `json:"wasm_congestion_max_factor,omitempty"`

	// WasmCongestionTargetBps is the budget use above which the factor
	// rises; zero means DefaultWasmCongestionTargetBps.
	WasmCongestionTargetBps uint32 `json:"wasm_congestion_target_bps,omitempty"`
}

// QueueWasmReserve is the queue's reserved share in force, in basis points.
func (p Params) QueueWasmReserve() uint32 {
	if p.QueueWasmReserveBps == 0 {
		return DefaultQueueWasmReserveBps
	}
	return p.QueueWasmReserveBps
}

// CongestionMaxFactor is the bound on the congestion factor in force.
func (p Params) CongestionMaxFactor() uint32 {
	if p.WasmCongestionMaxFactor == 0 {
		return DefaultWasmCongestionMaxFactor
	}
	return p.WasmCongestionMaxFactor
}

// CongestionTargetBps is the congestion target in force.
func (p Params) CongestionTargetBps() uint32 {
	if p.WasmCongestionTargetBps == 0 {
		return DefaultWasmCongestionTargetBps
	}
	return p.WasmCongestionTargetBps
}

// NextCongestionFactor is the factor (in millionths) the next block uses,
// and the height of the last block that exhausted its transaction pool,
// given this block's factor, its wasm budget use, whether its transaction
// pool was exhausted, its height and the last exhausted height:
//
//   - pool exhausted: up by half (and the height is recorded);
//   - above the target share: up by 1/8;
//   - below it: down by 1/9 (back by exactly one 1/8 step), but not within
//     CongestionHoldBlocks of the last exhausted block;
//   - at it: unchanged;
//
// always between CongestionFactorOne and the maximum. Integer arithmetic on
// stored values only, so every node computes the same factor.
func (p Params) NextCongestionFactor(cur, used, total uint64, exhausted bool, height, lastExhausted int64) (uint64, int64) {
	if cur < CongestionFactorOne {
		cur = CongestionFactorOne
	}
	maxF := uint64(p.CongestionMaxFactor()) * CongestionFactorOne
	target := uint64(p.CongestionTargetBps())
	next := cur
	switch {
	case total == 0:
	case exhausted:
		next = cur + cur/2
		lastExhausted = height
	case used*10_000 > total*target:
		next = cur + cur/8
	case used*10_000 < total*target:
		if height-lastExhausted > CongestionHoldBlocks {
			next = cur - cur/9
		}
	}
	if next < CongestionFactorOne {
		next = CongestionFactorOne
	}
	if next > maxF {
		next = maxF
	}
	return next, lastExhausted
}

// TxWasmShareBps is the per-transaction share of the block wasm budget in
// force, in basis points.
func (p Params) TxWasmShareBps() uint32 {
	if p.MaxTxWasmShareBps == 0 {
		return DefaultMaxTxWasmShareBps
	}
	return p.MaxTxWasmShareBps
}

// BlockLevelWasmGasLimit is the bound on one contract call made outside a
// transaction in force.
func (p Params) BlockLevelWasmGasLimit() uint64 {
	if p.MaxBlockLevelWasmGas == 0 {
		return DefaultMaxBlockLevelWasmGas
	}
	return p.MaxBlockLevelWasmGas
}

// BlockWasmGasLimit is the per-block CosmWasm execution budget in force.
func (p Params) BlockWasmGasLimit() uint64 {
	if p.MaxBlockWasmGas == 0 {
		return DefaultMaxBlockWasmGas
	}
	return p.MaxBlockWasmGas
}

// CosmosTxGasLimit is the per-transaction gas bound in force.
func (p Params) CosmosTxGasLimit() uint64 {
	if p.MaxCosmosTxGas == 0 {
		return DefaultMaxCosmosTxGas
	}
	return p.MaxCosmosTxGas
}

// RecordRetention is the retention of finished queued call records in force.
func (p Params) RecordRetention() uint64 {
	if p.RecordRetentionBlocks == 0 {
		return DefaultRecordRetentionBlocks
	}
	return p.RecordRetentionBlocks
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
	if p.MaxBlockWasmGas != 0 && (p.MaxBlockWasmGas < MinMaxBlockWasmGas || p.MaxBlockWasmGas > MaxMaxBlockWasmGas) {
		return fmt.Errorf("max block wasm gas %d is outside %d..%d", p.MaxBlockWasmGas, MinMaxBlockWasmGas, MaxMaxBlockWasmGas)
	}
	if p.MaxCosmosTxGas != 0 && (p.MaxCosmosTxGas < MinMaxCosmosTxGas || p.MaxCosmosTxGas > MaxMaxCosmosTxGas) {
		return fmt.Errorf("max cosmos tx gas %d is outside %d..%d", p.MaxCosmosTxGas, MinMaxCosmosTxGas, MaxMaxCosmosTxGas)
	}
	if p.QueueWasmReserveBps > 9_000 {
		return fmt.Errorf("queue wasm reserve %d bps is above 9000", p.QueueWasmReserveBps)
	}
	if p.WasmCongestionMaxFactor > 10_000 {
		return fmt.Errorf("wasm congestion max factor %d is above 10000", p.WasmCongestionMaxFactor)
	}
	if p.WasmCongestionTargetBps > 10_000 {
		return fmt.Errorf("wasm congestion target %d bps is above 10000", p.WasmCongestionTargetBps)
	}
	if p.MaxTxWasmShareBps > 10_000 {
		return fmt.Errorf("max tx wasm share %d bps is above 10000", p.MaxTxWasmShareBps)
	}
	if p.MaxBlockLevelWasmGas != 0 && (p.MaxBlockLevelWasmGas < MinMaxBlockLevelWasmGas || p.MaxBlockLevelWasmGas > MaxMaxBlockLevelWasmGas) {
		return fmt.Errorf("max block-level wasm gas %d is outside %d..%d", p.MaxBlockLevelWasmGas, MinMaxBlockLevelWasmGas, MaxMaxBlockLevelWasmGas)
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
