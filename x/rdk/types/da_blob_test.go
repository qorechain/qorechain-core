package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDABlobStruct(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	blob := DABlob{
		RollupID:   "test-rollup",
		BlobIndex:  1,
		Data:       []byte("hello world"),
		Commitment: []byte("commit123"),
		Height:     100,
		Namespace:  []byte("ns1"),
		StoredAt:   now,
		Pruned:     false,
	}

	data, err := json.Marshal(blob)
	if err != nil {
		t.Fatalf("failed to marshal DABlob: %v", err)
	}

	var decoded DABlob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal DABlob: %v", err)
	}

	if decoded.RollupID != blob.RollupID {
		t.Errorf("RollupID mismatch: expected %q, got %q", blob.RollupID, decoded.RollupID)
	}
	if decoded.BlobIndex != blob.BlobIndex {
		t.Errorf("BlobIndex mismatch: expected %d, got %d", blob.BlobIndex, decoded.BlobIndex)
	}
	if string(decoded.Data) != string(blob.Data) {
		t.Errorf("Data mismatch: expected %q, got %q", string(blob.Data), string(decoded.Data))
	}
	if decoded.Height != blob.Height {
		t.Errorf("Height mismatch: expected %d, got %d", blob.Height, decoded.Height)
	}
	if decoded.Pruned != blob.Pruned {
		t.Errorf("Pruned mismatch: expected %v, got %v", blob.Pruned, decoded.Pruned)
	}
	if !decoded.StoredAt.Equal(blob.StoredAt) {
		t.Errorf("StoredAt mismatch: expected %v, got %v", blob.StoredAt, decoded.StoredAt)
	}
}

func TestDACommitmentStruct(t *testing.T) {
	commitment := DACommitment{
		RollupID:  "test-rollup",
		BlobIndex: 5,
		Backend:   DANative,
		Hash:      []byte("hash123"),
		Size:      1024,
		Confirmed: true,
	}

	data, err := json.Marshal(commitment)
	if err != nil {
		t.Fatalf("failed to marshal DACommitment: %v", err)
	}

	var decoded DACommitment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal DACommitment: %v", err)
	}

	if decoded.RollupID != commitment.RollupID {
		t.Errorf("RollupID mismatch: expected %q, got %q", commitment.RollupID, decoded.RollupID)
	}
	if decoded.BlobIndex != commitment.BlobIndex {
		t.Errorf("BlobIndex mismatch: expected %d, got %d", commitment.BlobIndex, decoded.BlobIndex)
	}
	if decoded.Backend != DANative {
		t.Errorf("Backend mismatch: expected %q, got %q", DANative, decoded.Backend)
	}
	if decoded.Size != commitment.Size {
		t.Errorf("Size mismatch: expected %d, got %d", commitment.Size, decoded.Size)
	}
	if decoded.Confirmed != commitment.Confirmed {
		t.Errorf("Confirmed mismatch: expected %v, got %v", commitment.Confirmed, decoded.Confirmed)
	}
}

func TestDABackendValuesExist(t *testing.T) {
	backends := map[DABackend]string{
		DANative:   "native",
		DACelestia: "celestia",
		DABoth:     "both",
	}
	if len(backends) != 3 {
		t.Fatalf("expected 3 DA backends, got %d", len(backends))
	}
	for b, expected := range backends {
		if string(b) != expected {
			t.Errorf("backend: expected %q, got %q", expected, string(b))
		}
	}
}
