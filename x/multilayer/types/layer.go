package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// LayerType defines the type of subsidiary layer in the QoreChain multi-layer architecture
type LayerType string

const (
	LayerTypeUnspecified LayerType = ""
	LayerTypeSidechain  LayerType = "sidechain" // Compute-heavy operations (DeFi, Gaming, IoT)
	LayerTypePaychain   LayerType = "paychain"  // High-frequency microtransactions
	LayerTypeRollup     LayerType = "rollup"    // Application-specific rollups (DeFi, Gaming, NFT, Enterprise)
)

// LayerStatus defines the lifecycle status of a layer
type LayerStatus string

const (
	LayerStatusUnspecified    LayerStatus = ""
	LayerStatusProposed       LayerStatus = "proposed"
	LayerStatusActive         LayerStatus = "active"
	LayerStatusSuspended      LayerStatus = "suspended"
	LayerStatusDecommissioned LayerStatus = "decommissioned"
)

// ValidStatusTransitions defines allowed status transitions for layer lifecycle management.
// Enforced by the keeper to prevent invalid state changes.
var ValidStatusTransitions = map[LayerStatus][]LayerStatus{
	LayerStatusProposed:  {LayerStatusActive, LayerStatusDecommissioned},
	LayerStatusActive:    {LayerStatusSuspended, LayerStatusDecommissioned},
	LayerStatusSuspended: {LayerStatusActive, LayerStatusDecommissioned},
	// Decommissioned is terminal — no transitions allowed
}

// IsValidTransition checks whether transitioning from the current status to newStatus is allowed
func IsValidTransition(current, newStatus LayerStatus) bool {
	allowed, ok := ValidStatusTransitions[current]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

// LayerConfig defines the configuration for a subsidiary layer
type LayerConfig struct {
	LayerID     string      `json:"layer_id"`
	LayerType   LayerType   `json:"layer_type"`
	Status      LayerStatus `json:"status"`
	ChainID     string      `json:"chain_id,omitempty"` // ICS chain ID for sidechains
	Description string      `json:"description"`

	// Performance parameters
	TargetBlockTimeMs        uint64 `json:"target_block_time_ms"`
	MaxTransactionsPerBlock  uint64 `json:"max_transactions_per_block"`
	MaxGasPerBlock           uint64 `json:"max_gas_per_block"`

	// Security parameters
	MinValidators            uint32 `json:"min_validators"`
	SettlementIntervalBlocks uint64 `json:"settlement_interval_blocks"` // How often to anchor state to main chain
	ChallengePeriodSeconds   uint64 `json:"challenge_period_seconds"`   // Dispute window for state transitions

	// Fee parameters
	BaseFeeMultiplier              string `json:"base_fee_multiplier"`                // Fee multiplier vs main chain (decimal string)
	CrossLayerFeeBundlingEnabled   bool   `json:"cross_layer_fee_bundling_enabled"`   // CLFB support

	// Capabilities
	SupportedVMTypes  []string `json:"supported_vm_types,omitempty"`  // ["cosmwasm", "evm"]
	SupportedDomains  []string `json:"supported_domains,omitempty"`  // ["defi", "gaming", "iot"]

	// Timestamps
	RegisteredAt *time.Time `json:"registered_at,omitempty"`
	LastAnchorAt *time.Time `json:"last_anchor_at,omitempty"`

	// Creator
	Creator string `json:"creator"` // bech32 address of layer creator
}

// StateAnchor represents a state root commitment from a subsidiary chain
// anchored to the QoreChain main chain via Hierarchical Commitment Schemes (HCS)
type StateAnchor struct {
	LayerID              string    `json:"layer_id"`
	LayerHeight          uint64    `json:"layer_height"`            // Block height on the subsidiary chain
	StateRoot            []byte    `json:"state_root"`              // Merkle root of subsidiary chain state
	ValidatorSetHash     []byte    `json:"validator_set_hash"`      // Hash of the validator set that signed
	MainChainHeight      uint64    `json:"main_chain_height"`       // Main chain height when anchored
	AnchoredAt           time.Time `json:"anchored_at"`
	PQCAggregateSignature []byte   `json:"pqc_aggregate_signature"` // PQC (Dilithium-5) aggregate sig
	TransactionCount     uint64    `json:"transaction_count"`       // Number of TXs since last anchor
	CompressedStateProof []byte    `json:"compressed_state_proof"`  // CST: Compressed state transition proof
}

// CrossLayerMessage represents a message routed between layers
type CrossLayerMessage struct {
	MessageID        string    `json:"message_id"`
	SourceLayer      string    `json:"source_layer"`      // Origin layer ID ("main" for main chain)
	DestinationLayer string    `json:"destination_layer"` // Target layer ID
	Payload          []byte    `json:"payload"`           // Serialized transaction or message
	Fees             sdk.Coins `json:"fees"`              // Pre-paid cross-layer fees
	Sender           string    `json:"sender"`            // bech32 address
	CreatedAt        time.Time `json:"created_at"`
	TimeoutHeight    uint64    `json:"timeout_height"`    // Expiry height on destination
	RoutingDecision  string    `json:"routing_decision"`  // QCAI routing rationale
}

// RoutingDecision captures the QCAI Router's layer selection logic
type RoutingDecision struct {
	TransactionHash     string        `json:"transaction_hash"`
	SelectedLayer       string        `json:"selected_layer"`
	Reason              string        `json:"reason"`                // Human-readable reason
	LayerScores         []*LayerScore `json:"layer_scores"`          // Scores per candidate layer
	EstimatedGasSavings uint64        `json:"estimated_gas_savings"` // Gas saved vs main chain execution
	EstimatedLatencyMs  uint64        `json:"estimated_latency_ms"`  // Estimated time to finality
}

// LayerScore represents QCAI's score for a candidate layer
type LayerScore struct {
	LayerID          string `json:"layer_id"`
	Score            string `json:"score"`            // Decimal string 0.0-1.0
	CongestionFactor string `json:"congestion_factor"` // Current congestion 0.0-1.0
	CapabilityMatch  string `json:"capability_match"`  // How well layer matches TX requirements 0.0-1.0
	CostFactor       string `json:"cost_factor"`       // Relative cost vs main chain 0.0-1.0
}
