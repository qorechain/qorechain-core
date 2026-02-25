package types

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

// SlashingRecord stores a single infraction event for progressive slashing.
type SlashingRecord struct {
	ValidatorAddr    string `json:"validator_addr"`
	InfractionHeight int64  `json:"infraction_height"`
	InfractionType   string `json:"infraction_type"`   // "double_sign", "downtime", "light_client_attack"
	SeverityFactor   string `json:"severity_factor"`   // LegacyDec
	Penalty          string `json:"penalty"`           // LegacyDec — computed penalty applied
}
