package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFairBlockConfigDefault(t *testing.T) {
	cfg := DefaultFairBlockConfig()

	if cfg.Enabled {
		t.Error("expected Enabled to be false by default")
	}
	if cfg.TIBEThreshold <= 0 {
		t.Error("expected TIBEThreshold to be positive")
	}
	if cfg.DecryptionDelay < 0 {
		t.Error("expected DecryptionDelay to be non-negative")
	}
	if cfg.MaxEncryptedSize <= 0 {
		t.Error("expected MaxEncryptedSize to be positive")
	}
}

func TestGenesisStateValidation(t *testing.T) {
	gs := DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Fatalf("expected DefaultGenesisState().Validate() to return nil, got: %v", err)
	}
}

func TestEncryptedTxStruct(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tx := EncryptedTx{
		ID:            "enc-tx-001",
		EncryptedData: []byte("encrypted-payload-bytes"),
		Sender:        "qor1sender",
		TargetHeight:  500,
		SubmittedAt:   now,
		DecryptedData: nil,
		Decrypted:     false,
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("failed to marshal EncryptedTx: %v", err)
	}

	var decoded EncryptedTx
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal EncryptedTx: %v", err)
	}

	if decoded.ID != tx.ID {
		t.Errorf("ID mismatch: expected %q, got %q", tx.ID, decoded.ID)
	}
	if decoded.Sender != tx.Sender {
		t.Errorf("Sender mismatch: expected %q, got %q", tx.Sender, decoded.Sender)
	}
	if decoded.TargetHeight != tx.TargetHeight {
		t.Errorf("TargetHeight mismatch: expected %d, got %d", tx.TargetHeight, decoded.TargetHeight)
	}
	if !decoded.SubmittedAt.Equal(tx.SubmittedAt) {
		t.Errorf("SubmittedAt mismatch: expected %v, got %v", tx.SubmittedAt, decoded.SubmittedAt)
	}
	if decoded.Decrypted != tx.Decrypted {
		t.Errorf("Decrypted mismatch: expected %v, got %v", tx.Decrypted, decoded.Decrypted)
	}
	if string(decoded.EncryptedData) != string(tx.EncryptedData) {
		t.Errorf("EncryptedData mismatch")
	}
}
