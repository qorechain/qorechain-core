package types

import (
	"encoding/json"
	"testing"
)

func TestDefaultGasAbstractionConfig(t *testing.T) {
	cfg := DefaultGasAbstractionConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if cfg.NativeDenom != "uqor" {
		t.Errorf("expected NativeDenom to be %q, got %q", "uqor", cfg.NativeDenom)
	}

	// Verify all 3 expected denoms are present.
	expectedDenoms := map[string]bool{
		"uqor":     false,
		"ibc/USDC": false,
		"ibc/ATOM": false,
	}
	for _, token := range cfg.AcceptedTokens {
		if _, ok := expectedDenoms[token.Denom]; ok {
			expectedDenoms[token.Denom] = true
		}
	}
	for denom, found := range expectedDenoms {
		if !found {
			t.Errorf("expected denom %q in AcceptedTokens, but not found", denom)
		}
	}
}

func TestGenesisStateValidation(t *testing.T) {
	gs := DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Fatalf("expected DefaultGenesisState().Validate() to return nil, got: %v", err)
	}
}

func TestAcceptedFeeTokenJson(t *testing.T) {
	token := AcceptedFeeToken{
		Denom:          "ibc/USDC",
		ConversionRate: "1.0",
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("failed to marshal AcceptedFeeToken: %v", err)
	}

	var decoded AcceptedFeeToken
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal AcceptedFeeToken: %v", err)
	}

	if decoded.Denom != token.Denom {
		t.Errorf("Denom mismatch: expected %q, got %q", token.Denom, decoded.Denom)
	}
	if decoded.ConversionRate != token.ConversionRate {
		t.Errorf("ConversionRate mismatch: expected %q, got %q", token.ConversionRate, decoded.ConversionRate)
	}
}
