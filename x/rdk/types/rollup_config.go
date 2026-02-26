package types

import (
	"fmt"
	"time"
)

// --- Enums ---

// RollupProfile identifies a preset rollup configuration template.
type RollupProfile string

const (
	ProfileDeFi       RollupProfile = "defi"
	ProfileGaming     RollupProfile = "gaming"
	ProfileNFT        RollupProfile = "nft"
	ProfileEnterprise RollupProfile = "enterprise"
	ProfileCustom     RollupProfile = "custom"
)

// SettlementMode defines how a rollup settles on the host chain.
type SettlementMode string

const (
	SettlementOptimistic SettlementMode = "optimistic" // Fraud proofs, configurable challenge window
	SettlementZK         SettlementMode = "zk"         // Validity proofs, instant finality on proof verification
	SettlementBased      SettlementMode = "based"      // L1-sequenced: host chain proposers order rollup TXs
	SettlementSovereign  SettlementMode = "sovereign"  // Self-sequenced, no settlement on host chain
)

// DABackend defines which data availability backend to use.
type DABackend string

const (
	DANative   DABackend = "native"   // On-chain KVStore blob storage
	DACelestia DABackend = "celestia" // IBC to Celestia (stub in v1.3.0)
	DABoth     DABackend = "both"     // Native + Celestia redundancy
)

// RollupStatus represents the lifecycle state of a rollup.
type RollupStatus string

const (
	RollupStatusPending RollupStatus = "pending"
	RollupStatusActive  RollupStatus = "active"
	RollupStatusPaused  RollupStatus = "paused"
	RollupStatusStopped RollupStatus = "stopped"
)

// SequencerMode defines who sequences transactions for the rollup.
type SequencerMode string

const (
	SequencerDedicated SequencerMode = "dedicated" // Single operator sequences
	SequencerShared    SequencerMode = "shared"    // Shared sequencer set
	SequencerBased     SequencerMode = "based"     // L1 proposers sequence
)

// ProofSystem defines the proof mechanism used for settlement.
type ProofSystem string

const (
	ProofSystemFraud ProofSystem = "fraud" // Optimistic: interactive fraud proofs
	ProofSystemSNARK ProofSystem = "snark" // ZK: succinct proofs (Groth16, PLONK)
	ProofSystemSTARK ProofSystem = "stark" // ZK: transparent proofs (no trusted setup)
	ProofSystemNone  ProofSystem = "none"  // Based/Sovereign: no proofs needed
)

// --- Config Structs ---

// SequencerConfig defines the transaction sequencing configuration.
type SequencerConfig struct {
	Mode             SequencerMode `json:"mode"`
	SequencerAddress string        `json:"sequencer_address,omitempty"` // For dedicated: operator address
	SharedSetMinSize uint32        `json:"shared_set_min_size"`         // For shared: minimum sequencer set
	InclusionDelay   uint64        `json:"inclusion_delay"`             // For based: blocks before forced inclusion
	PriorityFeeShare string        `json:"priority_fee_share"`          // For based: % of priority fees to L1 proposer (e.g. "0.5")
}

// DefaultSequencerConfig returns a sensible default sequencer config.
func DefaultSequencerConfig() SequencerConfig {
	return SequencerConfig{
		Mode:             SequencerDedicated,
		SharedSetMinSize: 1,
		InclusionDelay:   10,
		PriorityFeeShare: "0.0",
	}
}

// ProofConfig defines the proof mechanism configuration.
type ProofConfig struct {
	System             ProofSystem `json:"system"`
	VerifierAddress    string      `json:"verifier_address,omitempty"` // On-chain verifier contract (ZK)
	ChallengeWindowSec uint64      `json:"challenge_window_sec"`       // Fraud proof window in seconds (Optimistic)
	ChallengeBond      int64       `json:"challenge_bond"`             // Bond required to submit challenge (uqor)
	MaxProofSize       uint64      `json:"max_proof_size"`             // Max proof bytes
	RecursionDepth     uint32      `json:"recursion_depth"`            // ZK: proof aggregation depth
}

// DefaultProofConfig returns a sensible default proof config.
func DefaultProofConfig() ProofConfig {
	return ProofConfig{
		System:             ProofSystemFraud,
		ChallengeWindowSec: 604800,     // 7 days
		ChallengeBond:      1000000000, // 1000 QOR in uqor
		MaxProofSize:       1048576,    // 1 MB
		RecursionDepth:     1,
	}
}

// RollupGasConfig defines the gas model for a rollup.
type RollupGasConfig struct {
	GasModel     string `json:"gas_model"`     // "eip1559", "flat", "standard", "subsidized"
	BaseGasPrice string `json:"base_gas_price"` // decimal string (e.g. "0.001")
	MaxGasLimit  uint64 `json:"max_gas_limit"`
}

// DefaultRollupGasConfig returns default gas configuration.
func DefaultRollupGasConfig() RollupGasConfig {
	return RollupGasConfig{
		GasModel:     "standard",
		BaseGasPrice: "0.001",
		MaxGasLimit:  10000000,
	}
}

// RollupConfig is the full configuration for an application-specific rollup.
type RollupConfig struct {
	RollupID        string          `json:"rollup_id"`
	Creator         string          `json:"creator"`
	Profile         RollupProfile   `json:"profile"`
	SettlementMode  SettlementMode  `json:"settlement_mode"`
	SequencerConfig SequencerConfig `json:"sequencer_config"`
	DABackend       DABackend       `json:"da_backend"`
	BlockTimeMs     uint64          `json:"block_time_ms"`
	MaxTxPerBlock   uint64          `json:"max_tx_per_block"`
	GasConfig       RollupGasConfig `json:"gas_config"`
	VMType          string          `json:"vm_type"` // "evm", "cosmwasm", "svm", "custom"
	ProofConfig     ProofConfig     `json:"proof_config"`
	Status          RollupStatus    `json:"status"`
	StakeAmount     int64           `json:"stake_amount"` // uqor
	LayerID         string          `json:"layer_id"`
	CreatedHeight   int64           `json:"created_height"`
	CreatedAt       time.Time       `json:"created_at"`
}

// Validate checks invariants on the RollupConfig.
func (c RollupConfig) Validate() error {
	if c.BlockTimeMs == 0 {
		return fmt.Errorf("block time must be positive")
	}
	if c.MaxTxPerBlock == 0 {
		return fmt.Errorf("max tx per block must be positive")
	}
	if c.StakeAmount <= 0 {
		return fmt.Errorf("stake amount must be positive")
	}

	// Based settlement requires based sequencer
	if c.SettlementMode == SettlementBased && c.SequencerConfig.Mode != SequencerBased {
		return fmt.Errorf("based settlement requires based sequencer mode")
	}

	// ZK settlement requires snark or stark proof system
	if c.SettlementMode == SettlementZK {
		if c.ProofConfig.System != ProofSystemSNARK && c.ProofConfig.System != ProofSystemSTARK {
			return fmt.Errorf("zk settlement requires snark or stark proof system")
		}
	}

	// Optimistic settlement requires fraud proof system
	if c.SettlementMode == SettlementOptimistic && c.ProofConfig.System != ProofSystemFraud {
		return fmt.Errorf("optimistic settlement requires fraud proof system")
	}

	// Sovereign settlement requires none proof system
	if c.SettlementMode == SettlementSovereign && c.ProofConfig.System != ProofSystemNone {
		return fmt.Errorf("sovereign settlement requires none proof system")
	}

	return nil
}
