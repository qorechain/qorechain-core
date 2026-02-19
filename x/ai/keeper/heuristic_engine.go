package keeper

import (
	"context"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// HeuristicEngine is a composite AIEngine using rule-based heuristics.
// It implements the types.AIEngine interface for the MVP.
type HeuristicEngine struct {
	router     *HeuristicRouter
	anomaly    *HeuristicAnomaly
	riskScorer *HeuristicRiskScorer
}

var _ types.AIEngine = (*HeuristicEngine)(nil)

// NewHeuristicEngine creates a new heuristic-based AI engine.
func NewHeuristicEngine(maxTxPerMinute int) *HeuristicEngine {
	return &HeuristicEngine{
		router:     NewHeuristicRouter(),
		anomaly:    NewHeuristicAnomaly(maxTxPerMinute),
		riskScorer: NewHeuristicRiskScorer(),
	}
}

func (e *HeuristicEngine) RouteTransaction(ctx context.Context, tx types.TransactionInfo) (*types.RoutingDecision, error) {
	return e.router.RouteTransaction(ctx, tx)
}

func (e *HeuristicEngine) DetectAnomaly(ctx context.Context, tx types.TransactionInfo, history []types.TransactionInfo) (*types.AnomalyResult, error) {
	return e.anomaly.DetectAnomaly(ctx, tx, history)
}

func (e *HeuristicEngine) ScoreContractRisk(ctx context.Context, code []byte, chain string) (*types.RiskScore, error) {
	return e.riskScorer.ScoreContractRisk(ctx, code, chain)
}
