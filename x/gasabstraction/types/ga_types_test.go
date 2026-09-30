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

	// Native fees only by default: the placeholder ibc/USDC and ibc/ATOM
	// entries were never real denoms, and conversion is opened by governance
	// once a real token and a pool exist (v3.2.2).
	if len(cfg.AcceptedTokens) != 1 || cfg.AcceptedTokens[0].Denom != "uqor" {
		t.Errorf("expected only uqor in AcceptedTokens, got %+v", cfg.AcceptedTokens)
	}
	if !cfg.BlockCap().IsZero() {
		t.Error("conversion must start closed (zero block cap)")
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

// TestGenesisValidateChecksRates pins QSR-2026-0262: genesis validation uses
// the full config rules, so a rate that is zero, negative or not a number is
// refused rather than only an empty one.
func TestGenesisValidateChecksRates(t *testing.T) {
	if err := DefaultGenesisState().Validate(); err != nil {
		t.Fatalf("default genesis must validate: %v", err)
	}
	for _, rate := range []string{"0", "-1", "abc", "", "0.0"} {
		gs := DefaultGenesisState()
		gs.Config.AcceptedTokens = append(gs.Config.AcceptedTokens, AcceptedFeeToken{Denom: "ibc/X", ConversionRate: rate})
		if err := gs.Validate(); err == nil {
			t.Errorf("rate %q validated", rate)
		}
	}
	gs := DefaultGenesisState()
	gs.Config.NativeDenom = ""
	if gs.Validate() == nil {
		t.Error("empty native denom validated")
	}
	// Every default accepted token parses to a positive rate.
	for _, tok := range DefaultGasAbstractionConfig().AcceptedTokens {
		one := GasAbstractionConfig{NativeDenom: "uqor", AcceptedTokens: []AcceptedFeeToken{tok}}
		if err := one.Validate(); err != nil {
			t.Errorf("default token %s: %v", tok.Denom, err)
		}
	}
}

// A config stored before v3.2.2 (the mainnet genesis shape, placeholders
// included) validates and serialises byte-identically.
func TestLegacyConfigBytesAndValidation(t *testing.T) {
	legacy := `{"enabled":true,"native_denom":"uqor","accepted_tokens":[{"denom":"uqor","conversion_rate":"1.0"},{"denom":"ibc/USDC","conversion_rate":"1.0"},{"denom":"ibc/ATOM","conversion_rate":"10.0"}]}`
	var cfg GasAbstractionConfig
	if err := json.Unmarshal([]byte(legacy), &cfg); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("legacy config refused: %v", err)
	}
	bz, _ := json.Marshal(cfg)
	if string(bz) != legacy {
		t.Fatalf("legacy config bytes changed:\n%s\n%s", bz, legacy)
	}
}

func TestPriceSourceValidation(t *testing.T) {
	base := func(tok AcceptedFeeToken) GasAbstractionConfig {
		c := DefaultGasAbstractionConfig()
		c.AcceptedTokens = append(c.AcceptedTokens, tok)
		return c
	}
	ok := []AcceptedFeeToken{
		{Denom: "ibc/A", ConversionRate: "2", PriceSource: PriceSourceStatic, RateExpiryHeight: 100},
		{Denom: "ibc/B", ConversionRate: "1", PriceSource: PriceSourceAMMTWAP, PoolID: 1, MinLiquidity: "1000", MaxDeviationBps: 200},
		{Denom: "ibc/C", ConversionRate: "1"}, // listed, no source: valid, cannot pay
	}
	for _, tok := range ok {
		if err := base(tok).Validate(); err != nil {
			t.Errorf("%s: %v", tok.Denom, err)
		}
	}
	bad := []AcceptedFeeToken{
		{Denom: "ibc/A", ConversionRate: "2", PriceSource: PriceSourceStatic},                                         // no expiry
		{Denom: "ibc/B", ConversionRate: "1", PriceSource: PriceSourceAMMTWAP, MinLiquidity: "1", MaxDeviationBps: 1}, // no pool
		{Denom: "ibc/B", ConversionRate: "1", PriceSource: PriceSourceAMMTWAP, PoolID: 1, MaxDeviationBps: 1},         // no depth
		{Denom: "ibc/B", ConversionRate: "1", PriceSource: PriceSourceAMMTWAP, PoolID: 1, MinLiquidity: "1"},          // no deviation bound
		{Denom: "ibc/D", ConversionRate: "1", PriceSource: "oracle"},
		{Denom: "uqor", ConversionRate: "1", PriceSource: PriceSourceStatic, RateExpiryHeight: 1},
	}
	for _, tok := range bad {
		if err := base(tok).Validate(); err == nil {
			t.Errorf("%+v validated", tok)
		}
	}
	c := DefaultGasAbstractionConfig()
	c.PremiumBps = MaxPremiumBps + 1
	if c.Validate() == nil {
		t.Error("premium above the cap validated")
	}
	c = DefaultGasAbstractionConfig()
	c.BlockConversionCap = "-5"
	if c.Validate() == nil {
		t.Error("negative cap validated")
	}
}
