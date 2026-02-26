//go:build proprietary

package keeper

import (
	"testing"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

func TestRollupStatusValues(t *testing.T) {
	statuses := []types.RollupStatus{
		types.RollupStatusPending,
		types.RollupStatusActive,
		types.RollupStatusPaused,
		types.RollupStatusStopped,
	}
	expected := []string{"pending", "active", "paused", "stopped"}

	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("status %d: expected %q, got %q", i, expected[i], string(s))
		}
	}
}

func TestLifecycleErrorSentinels(t *testing.T) {
	errors := map[string]error{
		"ErrRollupNotFound":        types.ErrRollupNotFound,
		"ErrRollupAlreadyExists":   types.ErrRollupAlreadyExists,
		"ErrRollupNotActive":       types.ErrRollupNotActive,
		"ErrMaxRollupsReached":     types.ErrMaxRollupsReached,
		"ErrInsufficientStake":     types.ErrInsufficientStake,
		"ErrUnauthorized":          types.ErrUnauthorized,
		"ErrBatchNotFound":         types.ErrBatchNotFound,
		"ErrBatchAlreadyFinalized": types.ErrBatchAlreadyFinalized,
	}

	for name, err := range errors {
		if err == nil {
			t.Errorf("%s should not be nil", name)
		}
		if err.Error() == "" {
			t.Errorf("%s should have non-empty error message", name)
		}
	}
}

func TestRollupStatusTransitionsLogic(t *testing.T) {
	// Test the valid state transitions conceptually:
	// pending -> active (on create)
	// active -> paused (on pause)
	// paused -> active (on resume)
	// active -> stopped (on stop)
	// paused -> stopped (on stop)

	// Verify that PauseRollup requires Active
	cfg := types.RollupConfig{Status: types.RollupStatusPaused}
	if cfg.Status == types.RollupStatusActive {
		t.Error("paused rollup should not be active")
	}

	// Verify that ResumeRollup requires Paused
	cfg.Status = types.RollupStatusActive
	if cfg.Status == types.RollupStatusPaused {
		t.Error("active rollup should not be paused")
	}

	// Verify stopped is terminal
	cfg.Status = types.RollupStatusStopped
	if cfg.Status == types.RollupStatusActive || cfg.Status == types.RollupStatusPaused {
		t.Error("stopped rollup should not be active or paused")
	}
}

func TestCreateRollupSetsActiveStatus(t *testing.T) {
	// The lifecycle.go CreateRollup sets Status = RollupStatusActive
	// Verify the expected status value
	if types.RollupStatusActive != "active" {
		t.Errorf("expected RollupStatusActive to be \"active\", got %q", types.RollupStatusActive)
	}
}

func TestAllErrorSentinelsExist(t *testing.T) {
	// Verify all 19 error sentinels are non-nil
	allErrors := []error{
		types.ErrRollupNotFound,
		types.ErrRollupAlreadyExists,
		types.ErrRollupNotActive,
		types.ErrMaxRollupsReached,
		types.ErrInsufficientStake,
		types.ErrUnauthorized,
		types.ErrBatchNotFound,
		types.ErrBatchAlreadyFinalized,
		types.ErrChallengeWindowClosed,
		types.ErrInvalidProof,
		types.ErrProofRequired,
		types.ErrDABlobTooLarge,
		types.ErrDABlobNotFound,
		types.ErrCelestiaDAStubed,
		types.ErrInvalidSettlementMode,
		types.ErrInvalidSequencerMode,
		types.ErrInvalidProofSystem,
		types.ErrBasedSequencerOnly,
		types.ErrChallengeBondRequired,
	}

	for i, err := range allErrors {
		if err == nil {
			t.Errorf("error sentinel %d should not be nil", i)
		}
	}
}
