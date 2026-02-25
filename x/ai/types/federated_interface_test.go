package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFederatedRoundState_Constants(t *testing.T) {
	states := []FederatedRoundState{
		FederatedRoundPending,
		FederatedRoundTraining,
		FederatedRoundAggregating,
		FederatedRoundComplete,
		FederatedRoundFailed,
	}

	seen := make(map[FederatedRoundState]bool)
	for _, s := range states {
		if s == "" {
			t.Error("round state constant should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate round state constant: %s", s)
		}
		seen[s] = true
	}
}

func TestFederatedUpdate_JSONMarshal(t *testing.T) {
	update := FederatedUpdate{
		NodeID:      "qor1abc123",
		Round:       42,
		Gradients:   []byte{0x01, 0x02, 0x03, 0x04},
		SampleCount: 1000,
		Loss:        0.0523,
		Metrics:     []byte(`{"accuracy": 0.95}`),
		Timestamp:   time.Date(2026, 2, 25, 14, 0, 0, 0, time.UTC),
		Signature:   []byte{0xAA, 0xBB, 0xCC},
	}

	bz, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal FederatedUpdate: %v", err)
	}

	var decoded FederatedUpdate
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("failed to unmarshal FederatedUpdate: %v", err)
	}

	if decoded.NodeID != update.NodeID {
		t.Errorf("NodeID mismatch: %s != %s", decoded.NodeID, update.NodeID)
	}
	if decoded.Round != 42 {
		t.Errorf("Round mismatch: %d != 42", decoded.Round)
	}
	if decoded.SampleCount != 1000 {
		t.Errorf("SampleCount mismatch: %d != 1000", decoded.SampleCount)
	}
	if decoded.Loss != update.Loss {
		t.Errorf("Loss mismatch: %f != %f", decoded.Loss, update.Loss)
	}
}

func TestFederatedRoundConfig_JSONMarshal(t *testing.T) {
	config := FederatedRoundConfig{
		MinParticipants:   5,
		MaxParticipants:   100,
		RoundTimeout:      300,
		AggregationMethod: "fedavg",
		LearningRate:      0.001,
		ClippingNorm:      1.0,
		NoiseMultiplier:   0.1,
	}

	bz, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal FederatedRoundConfig: %v", err)
	}

	var decoded FederatedRoundConfig
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("failed to unmarshal FederatedRoundConfig: %v", err)
	}

	if decoded.MinParticipants != 5 {
		t.Errorf("MinParticipants mismatch: %d != 5", decoded.MinParticipants)
	}
	if decoded.AggregationMethod != "fedavg" {
		t.Errorf("AggregationMethod mismatch: %s != fedavg", decoded.AggregationMethod)
	}
}

func TestFederatedRoundStatus_JSONMarshal(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(5 * time.Minute)

	status := FederatedRoundStatus{
		Round: 10,
		State: FederatedRoundComplete,
		Config: FederatedRoundConfig{
			MinParticipants:   3,
			MaxParticipants:   50,
			AggregationMethod: "scaffold",
		},
		TotalParticipants: 25,
		UpdatesReceived:   25,
		AverageLoss:       0.032,
		GlobalModelHash:   make([]byte, 32),
		StartedAt:         now,
		CompletedAt:       &completed,
	}

	bz, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal FederatedRoundStatus: %v", err)
	}

	var decoded FederatedRoundStatus
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("failed to unmarshal FederatedRoundStatus: %v", err)
	}

	if decoded.State != FederatedRoundComplete {
		t.Errorf("State mismatch: %s != complete", decoded.State)
	}
	if decoded.TotalParticipants != 25 {
		t.Errorf("TotalParticipants mismatch: %d != 25", decoded.TotalParticipants)
	}
	if decoded.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}
}

func TestFederatedGlobalModel_JSONMarshal(t *testing.T) {
	model := FederatedGlobalModel{
		Round:     5,
		ModelHash: make([]byte, 32),
		Weights:   []byte{0x01, 0x02},
		Timestamp: time.Now().UTC(),
	}

	bz, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("failed to marshal FederatedGlobalModel: %v", err)
	}

	var decoded FederatedGlobalModel
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("failed to unmarshal FederatedGlobalModel: %v", err)
	}

	if decoded.Round != 5 {
		t.Errorf("Round mismatch: %d != 5", decoded.Round)
	}
}
