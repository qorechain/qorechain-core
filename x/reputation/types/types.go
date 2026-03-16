package types

import "time"

// ReputationParams holds the weights for the reputation formula.
// R_i = α·S_i + β·P_i + γ·C_i + δ·T_i
type ReputationParams struct {
	Alpha    float64 `json:"alpha"`     // Stake weight
	Beta     float64 `json:"beta"`      // Performance weight
	Gamma    float64 `json:"gamma"`     // Contribution weight
	Delta    float64 `json:"delta"`     // Time weight
	Lambda   float64 `json:"lambda"`    // Decay constant (blocks)
	MinScore float64 `json:"min_score"` // Minimum reputation threshold
}

// DefaultReputationParams returns default parameters per whitepaper.
func DefaultReputationParams() ReputationParams {
	return ReputationParams{
		Alpha:    0.30,
		Beta:     0.35,
		Gamma:    0.20,
		Delta:    0.15,
		Lambda:   1000.0,
		MinScore: 0.1,
	}
}

// ValidatorReputation tracks reputation data for a single validator.
type ValidatorReputation struct {
	Address            string  `json:"address"`
	StakeScore         float64 `json:"stake_score"`
	PerformanceScore   float64 `json:"performance_score"`
	ContributionScore  float64 `json:"contribution_score"`
	TimeScore          float64 `json:"time_score"`
	CompositeScore     float64 `json:"composite_score"`
	LastUpdatedHeight  int64   `json:"last_updated_height"`
	UptimeBlocks       uint64  `json:"uptime_blocks"`
	ProposedBlocks     uint64  `json:"proposed_blocks"`
	MissedBlocks       uint64  `json:"missed_blocks"`
	SlashingEvents     uint64  `json:"slashing_events"`
	CommunityVotes     int64   `json:"community_votes"`
	JoinedAtHeight     int64   `json:"joined_at_height"`
}

// HistoricalScore records reputation for a specific block height.
type HistoricalScore struct {
	Height      int64     `json:"height"`
	Score       float64   `json:"score"`
	Timestamp   time.Time `json:"timestamp"`
	StakeComp   float64   `json:"stake_comp"`
	PerfComp    float64   `json:"perf_comp"`
	ContribComp float64   `json:"contrib_comp"`
	TimeComp    float64   `json:"time_comp"`
}
