package app

// LaneConfig defines the configuration for a single transaction lane.
// Lanes provide transaction prioritization in the mempool and block building.
// v1.2.0: Configuration-only — actual mempool ordering via PrepareProposal/ProcessProposal
// is a future milestone.
type LaneConfig struct {
	Name          string  `json:"name"`
	Priority      int     `json:"priority"`        // Higher = processed first (0-100)
	MaxBlockSpace float64 `json:"max_block_space"` // Fraction of block space (0.0-1.0)
	Description   string  `json:"description"`
}

// ConfigureLanes returns the ordered lane configurations.
// This is defined as a variable to allow override by build tags.
var ConfigureLanes func() []LaneConfig
