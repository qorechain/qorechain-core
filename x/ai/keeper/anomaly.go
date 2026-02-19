package keeper

import (
	"context"
	"math"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// HeuristicAnomaly implements statistical anomaly detection.
type HeuristicAnomaly struct {
	maxTxPerMinute int
}

func NewHeuristicAnomaly(maxTxPerMinute int) *HeuristicAnomaly {
	return &HeuristicAnomaly{maxTxPerMinute: maxTxPerMinute}
}

func (h *HeuristicAnomaly) DetectAnomaly(_ context.Context, tx types.TransactionInfo, history []types.TransactionInfo) (*types.AnomalyResult, error) {
	result := &types.AnomalyResult{
		IsAnomalous: false,
		Score:       0.0,
		Action:      "allow",
		Confidence:  0.85,
	}

	var flags []string
	var scores []float64

	// Rule 1: Rate limiting — count recent TXs from same sender
	recentCount := 0
	for _, htx := range history {
		if htx.Sender == tx.Sender {
			recentCount++
		}
	}
	if recentCount > h.maxTxPerMinute {
		flags = append(flags, "high_frequency")
		scores = append(scores, 0.6)
	}

	// Rule 2: Amount anomaly — TX amount > 3σ from sender's historical mean
	if len(history) > 2 {
		var sum, sumSq float64
		var count int
		for _, htx := range history {
			if htx.Sender == tx.Sender && htx.Amount > 0 {
				v := float64(htx.Amount)
				sum += v
				sumSq += v * v
				count++
			}
		}
		if count > 2 {
			mean := sum / float64(count)
			variance := (sumSq / float64(count)) - (mean * mean)
			if variance > 0 {
				stddev := math.Sqrt(variance)
				deviation := math.Abs(float64(tx.Amount)-mean) / stddev
				if deviation > 3.0 {
					flags = append(flags, "unusual_amount")
					scores = append(scores, math.Min(deviation/5.0, 1.0))
				}
			}
		}
	}

	// Rule 3: New account large transfer
	senderHistory := 0
	for _, htx := range history {
		if htx.Sender == tx.Sender {
			senderHistory++
		}
	}
	if senderHistory < 3 && tx.Amount > 1_000_000_000 { // < 3 txs and > 1000 QOR
		flags = append(flags, "new_account_large_transfer")
		scores = append(scores, 0.5)
	}

	// Rule 4: Rapid sequential transfers to same receiver
	sameReceiverCount := 0
	for _, htx := range history {
		if htx.Sender == tx.Sender && htx.Receiver == tx.Receiver {
			sameReceiverCount++
		}
	}
	if sameReceiverCount > 3 {
		flags = append(flags, "rapid_sequential_transfers")
		scores = append(scores, 0.4)
	}

	// Aggregate score
	if len(scores) > 0 {
		var maxScore float64
		for _, s := range scores {
			if s > maxScore {
				maxScore = s
			}
		}
		result.Score = maxScore
		result.Flags = flags
		result.IsAnomalous = maxScore > 0.5

		if maxScore > 0.8 {
			result.Action = "reject"
		} else if maxScore > 0.5 {
			result.Action = "flag"
		}
	}

	return result, nil
}
