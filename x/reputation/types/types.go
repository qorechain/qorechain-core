package types

import (
	"time"

	sdkmath "cosmossdk.io/math"
)

// ReputationParams holds the weights for the reputation formula.
// R_i = α·S_i + β·P_i + γ·C_i + δ·T_i
type ReputationParams struct {
	Alpha    string `json:"alpha"`     // Stake weight
	Beta     string `json:"beta"`      // Performance weight
	Gamma    string `json:"gamma"`     // Contribution weight
	Delta    string `json:"delta"`     // Time weight
	Lambda   string `json:"lambda"`    // Decay constant (blocks)
	MinScore string `json:"min_score"` // Minimum reputation threshold
}

// DefaultReputationParams returns default parameters per whitepaper.
func DefaultReputationParams() ReputationParams {
	return ReputationParams{
		Alpha:    sdkmath.LegacyNewDecWithPrec(30, 2).String(), // 0.30
		Beta:     sdkmath.LegacyNewDecWithPrec(35, 2).String(), // 0.35
		Gamma:    sdkmath.LegacyNewDecWithPrec(20, 2).String(), // 0.20
		Delta:    sdkmath.LegacyNewDecWithPrec(15, 2).String(), // 0.15
		Lambda:   sdkmath.LegacyNewDec(1000).String(),          // 1000
		MinScore: sdkmath.LegacyNewDecWithPrec(1, 1).String(),  // 0.1
	}
}

// ParamAlpha returns the Alpha weight as a LegacyDec.
func (p ReputationParams) ParamAlpha() sdkmath.LegacyDec {
	d, err := sdkmath.LegacyNewDecFromStr(p.Alpha)
	if err != nil {
		return sdkmath.LegacyNewDecWithPrec(30, 2)
	}
	return d
}

// ParamBeta returns the Beta weight as a LegacyDec.
func (p ReputationParams) ParamBeta() sdkmath.LegacyDec {
	d, err := sdkmath.LegacyNewDecFromStr(p.Beta)
	if err != nil {
		return sdkmath.LegacyNewDecWithPrec(35, 2)
	}
	return d
}

// ParamGamma returns the Gamma weight as a LegacyDec.
func (p ReputationParams) ParamGamma() sdkmath.LegacyDec {
	d, err := sdkmath.LegacyNewDecFromStr(p.Gamma)
	if err != nil {
		return sdkmath.LegacyNewDecWithPrec(20, 2)
	}
	return d
}

// ParamDelta returns the Delta weight as a LegacyDec.
func (p ReputationParams) ParamDelta() sdkmath.LegacyDec {
	d, err := sdkmath.LegacyNewDecFromStr(p.Delta)
	if err != nil {
		return sdkmath.LegacyNewDecWithPrec(15, 2)
	}
	return d
}

// ParamLambda returns the Lambda decay constant as a LegacyDec.
func (p ReputationParams) ParamLambda() sdkmath.LegacyDec {
	d, err := sdkmath.LegacyNewDecFromStr(p.Lambda)
	if err != nil {
		return sdkmath.LegacyNewDec(1000)
	}
	return d
}

// ParamMinScore returns the MinScore as a LegacyDec.
func (p ReputationParams) ParamMinScore() sdkmath.LegacyDec {
	d, err := sdkmath.LegacyNewDecFromStr(p.MinScore)
	if err != nil {
		return sdkmath.LegacyNewDecWithPrec(1, 1)
	}
	return d
}

// ValidatorReputation tracks reputation data for a single validator.
type ValidatorReputation struct {
	Address           string `json:"address"`
	StakeScore        string `json:"stake_score"`
	PerformanceScore  string `json:"performance_score"`
	ContributionScore string `json:"contribution_score"`
	TimeScore         string `json:"time_score"`
	CompositeScore    string `json:"composite_score"`
	LastUpdatedHeight int64  `json:"last_updated_height"`
	UptimeBlocks      uint64 `json:"uptime_blocks"`
	ProposedBlocks    uint64 `json:"proposed_blocks"`
	MissedBlocks      uint64 `json:"missed_blocks"`
	SlashingEvents    uint64 `json:"slashing_events"`
	CommunityVotes    int64  `json:"community_votes"`
	JoinedAtHeight    int64  `json:"joined_at_height"`
}

// GetCompositeScoreDec returns the CompositeScore as a LegacyDec.
// Returns zero if the string is empty or invalid (backward compat with old float64 data).
func (v ValidatorReputation) GetCompositeScoreDec() sdkmath.LegacyDec {
	return parseDec(v.CompositeScore)
}

// GetStakeScoreDec returns the StakeScore as a LegacyDec.
func (v ValidatorReputation) GetStakeScoreDec() sdkmath.LegacyDec {
	return parseDec(v.StakeScore)
}

// GetPerformanceScoreDec returns the PerformanceScore as a LegacyDec.
func (v ValidatorReputation) GetPerformanceScoreDec() sdkmath.LegacyDec {
	return parseDec(v.PerformanceScore)
}

// GetContributionScoreDec returns the ContributionScore as a LegacyDec.
func (v ValidatorReputation) GetContributionScoreDec() sdkmath.LegacyDec {
	return parseDec(v.ContributionScore)
}

// GetTimeScoreDec returns the TimeScore as a LegacyDec.
func (v ValidatorReputation) GetTimeScoreDec() sdkmath.LegacyDec {
	return parseDec(v.TimeScore)
}

// parseDec parses a string to LegacyDec, returning zero on failure.
// This provides backward compatibility: old float64 JSON values like "0.5"
// are valid LegacyDec strings, so existing genesis data continues to work.
func parseDec(s string) sdkmath.LegacyDec {
	if s == "" {
		return sdkmath.LegacyZeroDec()
	}
	d, err := sdkmath.LegacyNewDecFromStr(s)
	if err != nil {
		return sdkmath.LegacyZeroDec()
	}
	return d
}

// HistoricalScore records reputation for a specific block height.
type HistoricalScore struct {
	Height      int64     `json:"height"`
	Score       string    `json:"score"`
	Timestamp   time.Time `json:"timestamp"`
	StakeComp   string    `json:"stake_comp"`
	PerfComp    string    `json:"perf_comp"`
	ContribComp string    `json:"contrib_comp"`
	TimeComp    string    `json:"time_comp"`
}
