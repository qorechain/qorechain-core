package types

import (
	"encoding/json"
	"testing"
)

func TestBatchStatusValues(t *testing.T) {
	statuses := []BatchStatus{BatchSubmitted, BatchChallenged, BatchFinalized, BatchRejected}
	expected := []string{"submitted", "challenged", "finalized", "rejected"}

	if len(statuses) != 4 {
		t.Fatalf("expected 4 batch statuses, got %d", len(statuses))
	}
	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("status %d: expected %q, got %q", i, expected[i], string(s))
		}
	}
}

func TestSettlementBatchJSONRoundtrip(t *testing.T) {
	batch := SettlementBatch{
		RollupID:      "test-rollup",
		BatchIndex:    42,
		StateRoot:     []byte("stateroot123"),
		PrevStateRoot: []byte("prevroot456"),
		TxCount:       100,
		DataHash:      []byte("datahash789"),
		ProofType:     ProofSystemFraud,
		Proof:         []byte("proof-data"),
		SequencerMode: SequencerDedicated,
		L1BlockRange:  [2]int64{1000, 1010},
		SubmittedAt:   500,
		FinalizedAt:   0,
		Status:        BatchSubmitted,
	}

	data, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("failed to marshal SettlementBatch: %v", err)
	}

	var decoded SettlementBatch
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SettlementBatch: %v", err)
	}

	if decoded.RollupID != batch.RollupID {
		t.Errorf("RollupID mismatch: expected %q, got %q", batch.RollupID, decoded.RollupID)
	}
	if decoded.BatchIndex != batch.BatchIndex {
		t.Errorf("BatchIndex mismatch: expected %d, got %d", batch.BatchIndex, decoded.BatchIndex)
	}
	if decoded.TxCount != batch.TxCount {
		t.Errorf("TxCount mismatch: expected %d, got %d", batch.TxCount, decoded.TxCount)
	}
	if decoded.Status != batch.Status {
		t.Errorf("Status mismatch: expected %q, got %q", batch.Status, decoded.Status)
	}
	if decoded.L1BlockRange[0] != 1000 || decoded.L1BlockRange[1] != 1010 {
		t.Errorf("L1BlockRange mismatch: expected [1000,1010], got %v", decoded.L1BlockRange)
	}
	if string(decoded.ProofType) != string(ProofSystemFraud) {
		t.Errorf("ProofType mismatch: expected %q, got %q", ProofSystemFraud, decoded.ProofType)
	}
}

func TestSettlementBatchFields(t *testing.T) {
	batch := SettlementBatch{}
	if batch.RollupID != "" {
		t.Error("expected zero-value RollupID to be empty string")
	}
	if batch.BatchIndex != 0 {
		t.Error("expected zero-value BatchIndex to be 0")
	}
	if batch.Status != "" {
		t.Error("expected zero-value Status to be empty string")
	}
	if batch.SubmittedAt != 0 {
		t.Error("expected zero-value SubmittedAt to be 0")
	}
	if batch.FinalizedAt != 0 {
		t.Error("expected zero-value FinalizedAt to be 0")
	}
}
