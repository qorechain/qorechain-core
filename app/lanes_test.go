package app

import (
	"testing"
)

func TestConfigureLanesNotNil(t *testing.T) {
	if ConfigureLanes == nil {
		t.Fatal("expected ConfigureLanes to be non-nil (should be set by init)")
	}

	lanes := ConfigureLanes()
	if len(lanes) == 0 {
		t.Fatal("expected ConfigureLanes() to return at least one lane")
	}
}

func TestConfigureLanesOrdering(t *testing.T) {
	if ConfigureLanes == nil {
		t.Fatal("expected ConfigureLanes to be non-nil")
	}

	lanes := ConfigureLanes()
	if len(lanes) == 0 {
		t.Fatal("expected at least one lane")
	}

	// Verify lanes are in descending priority order.
	for i := 1; i < len(lanes); i++ {
		if lanes[i].Priority > lanes[i-1].Priority {
			t.Errorf("lanes not in descending priority order: lane[%d]=%q (priority %d) > lane[%d]=%q (priority %d)",
				i, lanes[i].Name, lanes[i].Priority,
				i-1, lanes[i-1].Name, lanes[i-1].Priority)
		}
	}
}
