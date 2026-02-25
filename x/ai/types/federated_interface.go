package types

// Federated Learning interfaces for the AI module.
// These define the contract for future on-chain federated learning coordination,
// where validator nodes train local models and submit gradient updates that are
// aggregated into a global model without sharing raw training data.
//
// v1.1.0: Interface specification only — no implementation.

import (
	"time"
)

// FederatedRoundState tracks the lifecycle of a federated learning round.
type FederatedRoundState string

const (
	FederatedRoundPending    FederatedRoundState = "pending"     // Waiting for participants
	FederatedRoundTraining   FederatedRoundState = "training"    // Local training in progress
	FederatedRoundAggregating FederatedRoundState = "aggregating" // Collecting and merging updates
	FederatedRoundComplete   FederatedRoundState = "complete"    // Round finished
	FederatedRoundFailed     FederatedRoundState = "failed"      // Round failed (timeout/insufficient participants)
)

// FederatedUpdate represents a single node's contribution to a federated
// learning round. Contains compressed gradient updates (not raw data).
type FederatedUpdate struct {
	NodeID      string    `json:"node_id"`       // Validator address
	Round       uint64    `json:"round"`         // Federated round number
	Gradients   []byte    `json:"gradients"`     // Compressed gradient tensor (FlatBuffers/protobuf)
	SampleCount uint64    `json:"sample_count"`  // Number of local training samples
	Loss        float64   `json:"loss"`          // Local training loss
	Metrics     []byte    `json:"metrics,omitempty"` // Optional per-round metrics (JSON)
	Timestamp   time.Time `json:"timestamp"`
	Signature   []byte    `json:"signature"`     // PQC signature over (round || gradients || sample_count)
}

// FederatedRoundConfig defines parameters for a federated learning round.
type FederatedRoundConfig struct {
	MinParticipants   uint32        `json:"min_participants"`   // Minimum nodes required
	MaxParticipants   uint32        `json:"max_participants"`   // Maximum nodes accepted
	RoundTimeout      int64         `json:"round_timeout_sec"`  // Seconds before round expires
	AggregationMethod string        `json:"aggregation_method"` // "fedavg", "fedprox", "scaffold"
	LearningRate      float64       `json:"learning_rate"`
	ClippingNorm      float64       `json:"clipping_norm"`      // Gradient clipping for privacy
	NoiseMultiplier   float64       `json:"noise_multiplier"`   // Differential privacy noise (0 = disabled)
}

// FederatedRoundStatus provides a summary of a federated learning round.
type FederatedRoundStatus struct {
	Round             uint64              `json:"round"`
	State             FederatedRoundState `json:"state"`
	Config            FederatedRoundConfig `json:"config"`
	TotalParticipants uint32              `json:"total_participants"`
	UpdatesReceived   uint32              `json:"updates_received"`
	AverageLoss       float64             `json:"average_loss"`
	GlobalModelHash   []byte              `json:"global_model_hash,omitempty"` // Set after aggregation
	StartedAt         time.Time           `json:"started_at"`
	CompletedAt       *time.Time          `json:"completed_at,omitempty"`
}

// FederatedGlobalModel represents the aggregated model state after a round.
type FederatedGlobalModel struct {
	Round     uint64    `json:"round"`
	ModelHash []byte    `json:"model_hash"` // Content-addressable hash of weights
	Weights   []byte    `json:"weights"`    // Serialized model weights
	Timestamp time.Time `json:"timestamp"`
}

// FederatedCoordinator orchestrates federated learning rounds across validator
// nodes. The coordinator runs on-chain (or as a privileged sidecar) and manages
// round lifecycle, update collection, and secure aggregation.
type FederatedCoordinator interface {
	// StartRound initiates a new federated learning round with the given config.
	// Returns the round number.
	StartRound(config FederatedRoundConfig) (uint64, error)

	// SubmitUpdate records a node's gradient update for the current round.
	// The update's PQC signature is verified before acceptance.
	SubmitUpdate(update FederatedUpdate) error

	// AggregateUpdates combines all submitted updates for a round into a
	// global model update using the configured aggregation method.
	// Returns the updated global model.
	AggregateUpdates(round uint64) (*FederatedGlobalModel, error)

	// GetRoundStatus returns the current status of a federated learning round.
	GetRoundStatus(round uint64) (*FederatedRoundStatus, error)

	// GetGlobalModel returns the latest aggregated global model.
	GetGlobalModel() (*FederatedGlobalModel, error)

	// GetCurrentRound returns the active round number (0 if none active).
	GetCurrentRound() uint64
}
