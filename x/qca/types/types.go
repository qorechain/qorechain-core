package types

// QCAConfig holds configuration for the QCA consensus module.
type QCAConfig struct {
	UseReputationWeighting bool    `json:"use_reputation_weighting"`
	MinReputationScore     float64 `json:"min_reputation_score"`
}

// DefaultQCAConfig returns the default configuration.
func DefaultQCAConfig() QCAConfig {
	return QCAConfig{
		UseReputationWeighting: true,
		MinReputationScore:     0.1,
	}
}

// QCAStats tracks module-level statistics.
type QCAStats struct {
	ProposerSelections    uint64 `json:"proposer_selections"`
	ReputationWeighted    uint64 `json:"reputation_weighted"`
	DefaultFallbacks      uint64 `json:"default_fallbacks"`
}

// ValidatorSelector is the interface for validator/proposer selection.
// MVP: HeuristicSelector. Future: AI-driven selection.
type ValidatorSelector interface {
	SelectProposer(validators []ValidatorInfo, scores map[string]float64, blockHash []byte, height int64) string
}

// ValidatorInfo holds minimal validator data for selection.
type ValidatorInfo struct {
	Address string  `json:"address"`
	Tokens  uint64  `json:"tokens"`
	Active  bool    `json:"active"`
}
