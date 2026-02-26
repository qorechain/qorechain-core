package types

import "testing"

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()

	if p.MaxRollups != 100 {
		t.Errorf("expected MaxRollups 100, got %d", p.MaxRollups)
	}
	if p.MinStakeForRollup != 10000000000 {
		t.Errorf("expected MinStakeForRollup 10000000000, got %d", p.MinStakeForRollup)
	}
	if p.RollupCreationBurnRate != "0.01" {
		t.Errorf("expected RollupCreationBurnRate \"0.01\", got %q", p.RollupCreationBurnRate)
	}
	if p.DefaultChallengeWindow != 604800 {
		t.Errorf("expected DefaultChallengeWindow 604800, got %d", p.DefaultChallengeWindow)
	}
	if p.MaxDABlobSize != 2097152 {
		t.Errorf("expected MaxDABlobSize 2097152, got %d", p.MaxDABlobSize)
	}
	if p.BlobRetentionBlocks != 432000 {
		t.Errorf("expected BlobRetentionBlocks 432000, got %d", p.BlobRetentionBlocks)
	}
	if p.MaxBatchesPerBlock != 10 {
		t.Errorf("expected MaxBatchesPerBlock 10, got %d", p.MaxBatchesPerBlock)
	}
}
