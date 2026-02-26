package types

import "testing"

func TestDefaultGenesisState(t *testing.T) {
	gs := DefaultGenesisState()
	if gs == nil {
		t.Fatal("DefaultGenesisState returned nil")
	}
	if err := gs.Validate(); err != nil {
		t.Fatalf("DefaultGenesisState().Validate() should return nil, got: %v", err)
	}
	if len(gs.Rollups) != 0 {
		t.Errorf("expected 0 rollups, got %d", len(gs.Rollups))
	}
	if len(gs.Batches) != 0 {
		t.Errorf("expected 0 batches, got %d", len(gs.Batches))
	}
}

func TestGenesisStateValidation(t *testing.T) {
	// Valid default
	gs := DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Errorf("expected default genesis to be valid, got: %v", err)
	}

	// Zero MaxRollups
	bad := *gs
	bad.Params.MaxRollups = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero MaxRollups")
	}

	// Zero MinStakeForRollup
	bad = *gs
	bad.Params.MinStakeForRollup = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero MinStakeForRollup")
	}

	// Negative MinStakeForRollup
	bad = *gs
	bad.Params.MinStakeForRollup = -1
	if err := bad.Validate(); err == nil {
		t.Error("expected error for negative MinStakeForRollup")
	}

	// Zero MaxDABlobSize
	bad = *gs
	bad.Params.MaxDABlobSize = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero MaxDABlobSize")
	}

	// Zero MaxBatchesPerBlock
	bad = *gs
	bad.Params.MaxBatchesPerBlock = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero MaxBatchesPerBlock")
	}

	// Invalid rollup in genesis
	bad = *gs
	bad.Rollups = []RollupConfig{{BlockTimeMs: 0, MaxTxPerBlock: 0, StakeAmount: 0}}
	if err := bad.Validate(); err == nil {
		t.Error("expected error for invalid rollup in genesis")
	}
}
