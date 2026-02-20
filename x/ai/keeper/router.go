//go:build proprietary

package keeper

import (
	"context"
	"fmt"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// HeuristicRouter implements rule-based transaction routing.
type HeuristicRouter struct{}

func NewHeuristicRouter() *HeuristicRouter {
	return &HeuristicRouter{}
}

func (h *HeuristicRouter) RouteTransaction(_ context.Context, tx types.TransactionInfo) (*types.RoutingDecision, error) {
	decision := &types.RoutingDecision{
		Priority:   0,
		Confidence: 0.9,
	}

	switch tx.TxType {
	case "delegate":
		decision.Reason = "staking tx routed to high-uptime validator"
		decision.Priority = 1
	case "contract_deploy":
		decision.Reason = "contract deploy routed to low-load validator"
		decision.Priority = 1
	default:
		if tx.Amount > 10_000_000_000 { // > 10,000 QOR (in uqor)
			decision.Reason = fmt.Sprintf("large value tx (%d uqor) routed to high-reputation validator", tx.Amount)
			decision.Priority = 1
		} else {
			decision.Reason = "default round-robin routing"
			decision.Priority = 0
		}
	}

	return decision, nil
}
