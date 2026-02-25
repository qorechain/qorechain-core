package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAbstractAccountConfigDefault(t *testing.T) {
	cfg := DefaultAbstractAccountConfig()

	if cfg.Enabled {
		t.Error("expected Enabled to be false by default")
	}
	if cfg.MaxSessionKeys <= 0 {
		t.Error("expected MaxSessionKeys to be positive")
	}
	if cfg.MaxSpendingRules <= 0 {
		t.Error("expected MaxSpendingRules to be positive")
	}
	if cfg.DefaultSessionTTL <= 0 {
		t.Error("expected DefaultSessionTTL to be positive")
	}
}

func TestSpendingRuleJson(t *testing.T) {
	rule := SpendingRule{
		ID:            "rule-001",
		DailyLimit:    1000000,
		PerTxLimit:    100000,
		AllowedDenoms: []string{"uqor", "ibc/USDC"},
		Enabled:       true,
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("failed to marshal SpendingRule: %v", err)
	}

	var decoded SpendingRule
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SpendingRule: %v", err)
	}

	if decoded.ID != rule.ID {
		t.Errorf("ID mismatch: expected %q, got %q", rule.ID, decoded.ID)
	}
	if decoded.DailyLimit != rule.DailyLimit {
		t.Errorf("DailyLimit mismatch: expected %d, got %d", rule.DailyLimit, decoded.DailyLimit)
	}
	if decoded.PerTxLimit != rule.PerTxLimit {
		t.Errorf("PerTxLimit mismatch: expected %d, got %d", rule.PerTxLimit, decoded.PerTxLimit)
	}
	if len(decoded.AllowedDenoms) != len(rule.AllowedDenoms) {
		t.Fatalf("AllowedDenoms length mismatch: expected %d, got %d", len(rule.AllowedDenoms), len(decoded.AllowedDenoms))
	}
	for i, d := range decoded.AllowedDenoms {
		if d != rule.AllowedDenoms[i] {
			t.Errorf("AllowedDenoms[%d] mismatch: expected %q, got %q", i, rule.AllowedDenoms[i], d)
		}
	}
	if decoded.Enabled != rule.Enabled {
		t.Errorf("Enabled mismatch: expected %v, got %v", rule.Enabled, decoded.Enabled)
	}
}

func TestSessionKeyExpiry(t *testing.T) {
	now := time.Now().UTC()

	// Session key that expired 1 hour ago.
	expired := SessionKey{
		Key:         "expired-key",
		Expiry:      now.Add(-1 * time.Hour),
		Permissions: []string{"send"},
		Label:       "test-expired",
		CreatedAt:   now.Add(-2 * time.Hour),
	}
	if !expired.IsExpired(now) {
		t.Error("expected session key with past expiry to be expired")
	}

	// Session key that expires 1 hour from now.
	valid := SessionKey{
		Key:         "valid-key",
		Expiry:      now.Add(1 * time.Hour),
		Permissions: []string{"send", "delegate"},
		Label:       "test-valid",
		CreatedAt:   now,
	}
	if valid.IsExpired(now) {
		t.Error("expected session key with future expiry to not be expired")
	}

	// Session key with expiry exactly at now -- not expired (After is strictly after).
	exact := SessionKey{
		Key:         "exact-key",
		Expiry:      now,
		Permissions: []string{"vote"},
		Label:       "test-exact",
		CreatedAt:   now,
	}
	if exact.IsExpired(now) {
		t.Error("expected session key with expiry == now to not be expired (time.After is strict)")
	}
}

func TestGenesisStateValidation(t *testing.T) {
	gs := DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Fatalf("expected DefaultGenesisState().Validate() to return nil, got: %v", err)
	}
}
