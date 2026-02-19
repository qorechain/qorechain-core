package keeper

import (
	"context"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// HeuristicRiskScorer implements pattern-matching contract risk scoring.
type HeuristicRiskScorer struct{}

func NewHeuristicRiskScorer() *HeuristicRiskScorer {
	return &HeuristicRiskScorer{}
}

func (h *HeuristicRiskScorer) ScoreContractRisk(_ context.Context, code []byte, _ string) (*types.RiskScore, error) {
	result := &types.RiskScore{
		Score:          0.0,
		Severity:       "LOW",
		Recommendation: "deploy",
	}

	var issues []types.RiskIssue

	// Rule 1: Code size limits
	if len(code) > 500_000 { // 500KB
		issues = append(issues, types.RiskIssue{
			Code:        "LARGE_CONTRACT",
			Description: "contract code exceeds 500KB",
			Severity:    "MEDIUM",
		})
	}

	// Rule 2: Empty contract
	if len(code) == 0 {
		issues = append(issues, types.RiskIssue{
			Code:        "EMPTY_CONTRACT",
			Description: "contract has no code",
			Severity:    "HIGH",
		})
	}

	// Calculate aggregate score
	if len(issues) > 0 {
		var maxSev float64
		for _, issue := range issues {
			switch issue.Severity {
			case "LOW":
				if 0.2 > maxSev {
					maxSev = 0.2
				}
			case "MEDIUM":
				if 0.5 > maxSev {
					maxSev = 0.5
				}
			case "HIGH":
				if 0.8 > maxSev {
					maxSev = 0.8
				}
			case "CRITICAL":
				maxSev = 1.0
			}
		}
		result.Score = maxSev
		result.Issues = issues

		switch {
		case maxSev >= 0.8:
			result.Severity = "HIGH"
			result.Recommendation = "reject"
		case maxSev >= 0.5:
			result.Severity = "MEDIUM"
			result.Recommendation = "review"
		default:
			result.Severity = "LOW"
			result.Recommendation = "deploy"
		}
	}

	return result, nil
}
