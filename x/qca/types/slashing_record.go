package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// SlashingConfig holds progressive slashing parameters.
type SlashingConfig struct {
	BaseRate         string `json:"base_rate"`          // base slash rate (LegacyDec, default: "0.01" = 1%)
	EscalationFactor string `json:"escalation_factor"`  // progressive multiplier base (LegacyDec, default: "1.5")
	MaxPenalty       string `json:"max_penalty"`        // maximum slash per event (LegacyDec, default: "0.33" = 33%)
	DecayHalflife    uint64 `json:"decay_halflife"`     // blocks for half-life decay (default: 100000)
}

// DefaultSlashingConfig returns the default progressive slashing configuration.
func DefaultSlashingConfig() SlashingConfig {
	return SlashingConfig{
		BaseRate:         "0.01",
		EscalationFactor: "1.5",
		MaxPenalty:       "0.33",
		DecayHalflife:    100_000,
	}
}

// Validate checks that slashing config parameters are well-formed.
func (cfg SlashingConfig) Validate() error {
	baseRate, err := math.LegacyNewDecFromStr(cfg.BaseRate)
	if err != nil {
		return fmt.Errorf("invalid base_rate %q: %w", cfg.BaseRate, err)
	}
	if !baseRate.IsPositive() {
		return fmt.Errorf("base_rate must be positive, got %s", baseRate)
	}

	escalation, err := math.LegacyNewDecFromStr(cfg.EscalationFactor)
	if err != nil {
		return fmt.Errorf("invalid escalation_factor %q: %w", cfg.EscalationFactor, err)
	}
	if escalation.LT(math.LegacyOneDec()) {
		return fmt.Errorf("escalation factor must be >= 1.0, got %s", escalation)
	}

	maxPenalty, err := math.LegacyNewDecFromStr(cfg.MaxPenalty)
	if err != nil {
		return fmt.Errorf("invalid max_penalty %q: %w", cfg.MaxPenalty, err)
	}
	if !maxPenalty.IsPositive() {
		return fmt.Errorf("max_penalty must be positive, got %s", maxPenalty)
	}

	if cfg.DecayHalflife == 0 {
		return fmt.Errorf("decay_halflife must be > 0")
	}

	return nil
}

// SlashingRecord stores a single infraction event for progressive slashing.
type SlashingRecord struct {
	ValidatorAddr    string `json:"validator_addr"`
	InfractionHeight int64  `json:"infraction_height"`
	InfractionType   string `json:"infraction_type"`   // "double_sign", "downtime", "light_client_attack"
	SeverityFactor   string `json:"severity_factor"`   // LegacyDec
	Penalty          string `json:"penalty"`           // LegacyDec — computed penalty applied
}
