//go:build proprietary

package keeper

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// HeuristicRouter implements QCAI-powered transaction routing using rule-based scoring.
// CRITICAL: This will be replaced with a trained ML model once sufficient
// routing data has been collected from the testnet (target: 3-6 months of data).
// The QCAIRouterInterface is designed specifically for this swap.
type HeuristicRouter struct {
	// Current: rule-based weights for QCAI routing heuristics
	congestionWeight  float64 // 0.3
	capabilityWeight  float64 // 0.4
	costWeight        float64 // 0.2
	latencyWeight     float64 // 0.1
}

// NewHeuristicRouter creates a new QCAI heuristic router with default weights.
func NewHeuristicRouter() *HeuristicRouter {
	return &HeuristicRouter{
		congestionWeight:  0.3,
		capabilityWeight:  0.4,
		costWeight:        0.2,
		latencyWeight:     0.1,
	}
}

// RouteTransaction processes a routing request through the QCAI heuristic engine.
// It evaluates all active layers, scores them, and selects the optimal one.
func (k Keeper) RouteTransaction(ctx sdk.Context, msg *types.MsgRouteTransaction) (*types.MsgRouteTransactionResponse, error) {
	params := k.GetParams(ctx)

	// Check if QCAI routing is enabled
	if !params.RoutingEnabled {
		return nil, types.ErrRoutingDisabled
	}

	// Get all active layers
	activeLayers := k.getActiveLayers(ctx)

	// Score layers using the QCAI heuristic engine
	scores, err := k.router.ScoreLayers(ctx, msg.TransactionPayload, activeLayers)
	if err != nil {
		return nil, fmt.Errorf("QCAI router scoring failed: %w", err)
	}

	// Select optimal layer
	decision, err := k.router.SelectOptimalLayer(scores, msg.MaxLatencyMs, msg.MaxFee)
	if err != nil {
		return nil, fmt.Errorf("QCAI router selection failed: %w", err)
	}

	// Set transaction hash
	txHash := sha256.Sum256(msg.TransactionPayload)
	decision.TransactionHash = fmt.Sprintf("%x", txHash[:])

	// Check confidence threshold
	threshold, _ := strconv.ParseFloat(params.RoutingConfidenceThreshold, 64)
	if len(decision.LayerScores) > 0 {
		bestScore, _ := strconv.ParseFloat(decision.LayerScores[0].Score, 64)
		if bestScore < threshold {
			// Below confidence threshold — default to main chain
			decision.SelectedLayer = "main"
			decision.Reason = fmt.Sprintf("QCAI confidence %.2f below threshold %.2f; defaulting to main chain", bestScore, threshold)
		}
	}

	// Apply preferred layer hint if provided
	if msg.PreferredLayer != "" && decision.SelectedLayer == "main" {
		for _, score := range decision.LayerScores {
			if score.LayerID == msg.PreferredLayer {
				s, _ := strconv.ParseFloat(score.Score, 64)
				if s >= threshold*0.8 { // Accept preferred layer with 80% threshold
					decision.SelectedLayer = msg.PreferredLayer
					decision.Reason = fmt.Sprintf("preferred layer %s accepted (score %.2f)", msg.PreferredLayer, s)
				}
				break
			}
		}
	}

	// Update routing statistics
	stats := k.getRoutingStats(ctx)
	stats.TotalRouted++
	switch decision.SelectedLayer {
	case "main":
		stats.RoutedToMain++
	default:
		// Check if it's a sidechain or paychain
		layer, err := k.GetLayer(ctx, decision.SelectedLayer)
		if err == nil {
			switch layer.LayerType {
			case types.LayerTypeSidechain:
				stats.RoutedToSidechains++
			case types.LayerTypePaychain:
				stats.RoutedToPaychains++
			}
		}
	}
	stats.TotalGasSavings += decision.EstimatedGasSavings
	stats.TotalLatencyImprovement += decision.EstimatedLatencyMs
	_ = k.setRoutingStats(ctx, stats)

	// Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeTransactionRouted,
		sdk.NewAttribute(types.AttributeKeySelectedLayer, decision.SelectedLayer),
		sdk.NewAttribute(types.AttributeKeyRoutingReason, decision.Reason),
		sdk.NewAttribute(types.AttributeKeyGasSavings, fmt.Sprintf("%d", decision.EstimatedGasSavings)),
		sdk.NewAttribute(types.AttributeKeyLatencyMs, fmt.Sprintf("%d", decision.EstimatedLatencyMs)),
		sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
	))

	k.logger.Info("transaction routed by QCAI",
		"selected_layer", decision.SelectedLayer,
		"reason", decision.Reason,
		"gas_savings", decision.EstimatedGasSavings,
		"sender", msg.Sender,
	)

	return &types.MsgRouteTransactionResponse{
		Decision: decision,
	}, nil
}

// SimulateRoute simulates QCAI routing for a transaction without executing it.
func (k Keeper) SimulateRoute(ctx sdk.Context, payload []byte, maxLatency uint64, maxFee string) (*types.RoutingDecision, error) {
	params := k.GetParams(ctx)
	if !params.RoutingEnabled {
		return &types.RoutingDecision{
			SelectedLayer: "main",
			Reason:        "QCAI routing is disabled",
		}, nil
	}

	activeLayers := k.getActiveLayers(ctx)
	scores, err := k.router.ScoreLayers(ctx, payload, activeLayers)
	if err != nil {
		return nil, err
	}

	return k.router.SelectOptimalLayer(scores, maxLatency, maxFee)
}

// ---- HeuristicRouter Implementation ----

// ScoreLayers evaluates all active layers for a given transaction using QCAI heuristics.
// Returns scores per layer based on congestion, capability match, cost.
func (r *HeuristicRouter) ScoreLayers(_ sdk.Context, txPayload []byte, activeLayers []*types.LayerConfig) ([]*types.LayerScore, error) {
	var scores []*types.LayerScore

	// Always include main chain as a candidate
	scores = append(scores, &types.LayerScore{
		LayerID:          "main",
		Score:            "0.5", // Baseline score for main chain
		CongestionFactor: "0.3", // Moderate congestion assumed
		CapabilityMatch:  "1.0", // Main chain can handle everything
		CostFactor:       "1.0", // Base cost
	})

	// Score each active layer
	for _, layer := range activeLayers {
		congestion := r.estimateCongestion(layer)
		capability := r.estimateCapabilityMatch(txPayload, layer)
		cost := r.estimateCostFactor(layer)
		latency := r.estimateLatencyFactor(layer)

		// Weighted score: higher is better
		// Congestion: lower congestion = higher score (inverted)
		// Capability: higher match = higher score
		// Cost: lower cost = higher score (inverted)
		// Latency: lower latency = higher score (inverted)
		score := r.congestionWeight*(1.0-congestion) +
			r.capabilityWeight*capability +
			r.costWeight*(1.0-cost) +
			r.latencyWeight*(1.0-latency)

		scores = append(scores, &types.LayerScore{
			LayerID:          layer.LayerID,
			Score:            formatFloat(score),
			CongestionFactor: formatFloat(congestion),
			CapabilityMatch:  formatFloat(capability),
			CostFactor:       formatFloat(cost),
		})
	}

	return scores, nil
}

// SelectOptimalLayer picks the best layer based on scores and constraints.
func (r *HeuristicRouter) SelectOptimalLayer(scores []*types.LayerScore, maxLatency uint64, maxFee string) (*types.RoutingDecision, error) {
	if len(scores) == 0 {
		return &types.RoutingDecision{
			SelectedLayer: "main",
			Reason:        "no candidate layers available",
		}, nil
	}

	// Find the highest scoring layer
	bestIdx := 0
	bestScore := 0.0
	for i, score := range scores {
		s, _ := strconv.ParseFloat(score.Score, 64)
		if s > bestScore {
			bestScore = s
			bestIdx = i
		}
	}

	selected := scores[bestIdx]

	// Estimate gas savings vs main chain
	var gasSavings uint64
	var latencyMs uint64
	if selected.LayerID != "main" {
		cost, _ := strconv.ParseFloat(selected.CostFactor, 64)
		gasSavings = uint64((1.0 - cost) * 100000) // Estimated savings in gas units
		congestion, _ := strconv.ParseFloat(selected.CongestionFactor, 64)
		latencyMs = uint64((1.0 - congestion) * 5000) // Estimated latency improvement
	}

	return &types.RoutingDecision{
		SelectedLayer:       selected.LayerID,
		Reason:              fmt.Sprintf("QCAI heuristic selected %s (score: %s)", selected.LayerID, selected.Score),
		LayerScores:         scores,
		EstimatedGasSavings: gasSavings,
		EstimatedLatencyMs:  latencyMs,
	}, nil
}

// ---- Heuristic Scoring Functions ----

// estimateCongestion estimates the congestion level of a layer (0.0-1.0).
// Uses recent anchor transaction counts as a proxy for layer load.
func (r *HeuristicRouter) estimateCongestion(layer *types.LayerConfig) float64 {
	// Heuristic: estimate from target block time (faster blocks = lower per-block congestion)
	if layer.TargetBlockTimeMs == 0 {
		return 0.5
	}
	// Faster block time = lower congestion (more throughput capacity)
	// Normalize: 1000ms = 0.5, 500ms = 0.25, 2000ms = 0.75
	return math.Min(1.0, float64(layer.TargetBlockTimeMs)/2000.0)
}

// estimateCapabilityMatch checks if TX type matches layer's supported domains (0.0-1.0).
func (r *HeuristicRouter) estimateCapabilityMatch(txPayload []byte, layer *types.LayerConfig) float64 {
	// Heuristic: classify TX by payload size
	// Small payloads (< 256 bytes) = likely microtransactions → paychains preferred
	// Large payloads (> 1024 bytes) = likely complex operations → sidechains preferred
	payloadSize := len(txPayload)

	if layer.LayerType == types.LayerTypePaychain {
		if payloadSize < 256 {
			return 0.9 // Excellent match for microtransactions
		}
		return 0.3 // Poor match for complex operations
	}

	if layer.LayerType == types.LayerTypeSidechain {
		if payloadSize > 1024 {
			return 0.9 // Excellent match for complex operations
		}
		if payloadSize > 256 {
			return 0.7 // Good match
		}
		return 0.4 // Moderate match
	}

	return 0.5 // Default
}

// estimateCostFactor estimates the relative cost vs main chain (0.0-1.0).
func (r *HeuristicRouter) estimateCostFactor(layer *types.LayerConfig) float64 {
	multiplier, err := strconv.ParseFloat(layer.BaseFeeMultiplier, 64)
	if err != nil || multiplier <= 0 {
		return 0.5
	}
	// Normalize: 0.01 = 0.01, 1.0 = 1.0
	return math.Min(1.0, multiplier)
}

// estimateLatencyFactor estimates the latency factor (0.0-1.0).
func (r *HeuristicRouter) estimateLatencyFactor(layer *types.LayerConfig) float64 {
	if layer.TargetBlockTimeMs == 0 {
		return 0.5
	}
	// Normalize: 500ms = 0.1, 1000ms = 0.2, 5000ms = 1.0
	return math.Min(1.0, float64(layer.TargetBlockTimeMs)/5000.0)
}

// formatFloat formats a float64 to a string with 2 decimal places.
func formatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
