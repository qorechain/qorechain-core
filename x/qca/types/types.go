package types

// QCAConfig holds configuration for the QCA consensus module.
type QCAConfig struct {
	UseReputationWeighting bool               `json:"use_reputation_weighting"`
	MinReputationScore     float64            `json:"min_reputation_score"`
	PoolConfig             PoolConfig         `json:"pool_config"`
	BondingCurveConfig     BondingCurveConfig `json:"bonding_curve_config"`
	SlashingConfig         SlashingConfig     `json:"slashing_config"`
	QDRWConfig             QDRWConfig         `json:"qdrw_config"`
}

// DefaultQCAConfig returns the default configuration.
func DefaultQCAConfig() QCAConfig {
	return QCAConfig{
		UseReputationWeighting: true,
		MinReputationScore:     0.1,
		PoolConfig:             DefaultPoolConfig(),
		BondingCurveConfig:     DefaultBondingCurveConfig(),
		SlashingConfig:         DefaultSlashingConfig(),
		QDRWConfig:             DefaultQDRWConfig(),
	}
}

// QCAStats tracks module-level statistics.
type QCAStats struct {
	ProposerSelections  uint64 `json:"proposer_selections"`
	ReputationWeighted  uint64 `json:"reputation_weighted"`
	DefaultFallbacks    uint64 `json:"default_fallbacks"`
	PoolClassifications uint64 `json:"pool_classifications"`
	SlashingEvents      uint64 `json:"slashing_events"`
	BondingCalculations uint64 `json:"bonding_calculations"`
}

// ValidatorSelector is the interface for validator/proposer selection.
// MVP: HeuristicSelector. Future: AI-driven selection.
type ValidatorSelector interface {
	SelectProposer(validators []ValidatorInfo, scores map[string]float64, blockHash []byte, height int64) (string, error)
}

// ValidatorInfo holds minimal validator data for selection.
type ValidatorInfo struct {
	Address string `json:"address"`
	Tokens  uint64 `json:"tokens"`
	Active  bool   `json:"active"`
}

// QDRWConfig holds QDRW (Quadratic Delegation with Reputation Weighting) parameters.
type QDRWConfig struct {
	Enabled          bool   `json:"enabled"`            // enable QDRW tally (default: false)
	XQOREMultiplier  string `json:"xqore_multiplier"`   // xQORE weight relative to staked tokens (LegacyDec, default: "2.0")
	RepMinMultiplier string `json:"rep_min_multiplier"` // minimum reputation multiplier (LegacyDec, default: "0.5")
	RepMaxMultiplier string `json:"rep_max_multiplier"` // maximum reputation multiplier (LegacyDec, default: "2.0")
}

// DefaultQDRWConfig returns the default QDRW configuration.
func DefaultQDRWConfig() QDRWConfig {
	return QDRWConfig{
		Enabled:          false,
		XQOREMultiplier:  "2.0",
		RepMinMultiplier: "0.5",
		RepMaxMultiplier: "2.0",
	}
}
