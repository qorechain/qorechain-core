//go:build proprietary

package keeper

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// NetworkOptimizer continuously monitors network performance and recommends
// parameter adjustments. Implements the whitepaper's reward function:
// R(s,a,s') = α·ΔPerformance + β·ΔLatency + γ·ΔEnergy - δ·StabilityPenalty
type NetworkOptimizer struct {
	stateMonitor    *NetworkStateMonitor
	paramController *ParameterController
}

// NewNetworkOptimizer creates a new network optimizer.
func NewNetworkOptimizer() *NetworkOptimizer {
	return &NetworkOptimizer{
		stateMonitor:    NewNetworkStateMonitor(),
		paramController: NewParameterController(),
	}
}

// Optimize analyzes current network state and returns parameter recommendations.
func (no *NetworkOptimizer) Optimize(_ context.Context) ([]types.NetworkRecommendation, error) {
	state := no.stateMonitor.CurrentState()
	return no.paramController.Recommend(state), nil
}

// RecordState feeds new network state data into the monitor.
func (no *NetworkOptimizer) RecordState(state types.NetworkState) {
	no.stateMonitor.Record(state)
}

// ---- Network State Monitor ----

// NetworkStateMonitor tracks network performance metrics over time.
type NetworkStateMonitor struct {
	mu      sync.RWMutex
	history []timestampedState
	maxSize int
}

type timestampedState struct {
	state     types.NetworkState
	timestamp time.Time
}

func NewNetworkStateMonitor() *NetworkStateMonitor {
	return &NetworkStateMonitor{
		history: make([]timestampedState, 0, 200),
		maxSize: 200,
	}
}

// Record adds a new state snapshot.
func (m *NetworkStateMonitor) Record(state types.NetworkState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, timestampedState{
		state:     state,
		timestamp: time.Now(),
	})

	if len(m.history) > m.maxSize {
		m.history = m.history[1:]
	}
}

// CurrentState returns the most recent state, or a zero state if none recorded.
func (m *NetworkStateMonitor) CurrentState() types.NetworkState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.history) == 0 {
		return types.NetworkState{}
	}
	return m.history[len(m.history)-1].state
}

// Trend computes the trend (change per minute) for a metric over recent history.
func (m *NetworkStateMonitor) Trend(metricFn func(types.NetworkState) float64) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.history) < 5 {
		return 0.0
	}

	recent := m.history[len(m.history)-5:]
	if recent[len(recent)-1].timestamp.Sub(recent[0].timestamp) == 0 {
		return 0.0
	}

	startVal := metricFn(recent[0].state)
	endVal := metricFn(recent[len(recent)-1].state)
	minutes := recent[len(recent)-1].timestamp.Sub(recent[0].timestamp).Minutes()
	if minutes == 0 {
		return 0.0
	}

	return (endVal - startVal) / minutes
}

// ---- Parameter Controller ----

// ParameterController generates governance parameter recommendations based on
// network state. Uses the whitepaper's reward function approach to evaluate
// suggested changes.
type ParameterController struct {
	// Reward function weights
	alpha float64 // Performance weight
	beta  float64 // Latency weight
	gamma float64 // Energy weight
	delta float64 // Stability penalty weight
}

func NewParameterController() *ParameterController {
	return &ParameterController{
		alpha: 0.35,
		beta:  0.30,
		gamma: 0.15,
		delta: 0.20,
	}
}

// Recommend generates parameter adjustment recommendations based on current state.
func (pc *ParameterController) Recommend(state types.NetworkState) []types.NetworkRecommendation {
	var recs []types.NetworkRecommendation

	// Recommendation 1: Block gas limit adjustment
	if rec := pc.recommendBlockGas(state); rec != nil {
		recs = append(recs, *rec)
	}

	// Recommendation 2: Minimum commission rate
	if rec := pc.recommendMinCommission(state); rec != nil {
		recs = append(recs, *rec)
	}

	// Recommendation 3: Max validators
	if rec := pc.recommendMaxValidators(state); rec != nil {
		recs = append(recs, *rec)
	}

	// Recommendation 4: Block time target
	if rec := pc.recommendBlockTime(state); rec != nil {
		recs = append(recs, *rec)
	}

	return recs
}

func (pc *ParameterController) recommendBlockGas(state types.NetworkState) *types.NetworkRecommendation {
	if state.BlockUtilization == 0 {
		return nil
	}

	// If blocks are consistently >80% full, suggest increasing gas limit
	if state.BlockUtilization > 0.8 {
		return &types.NetworkRecommendation{
			Parameter:      "max_block_gas",
			CurrentValue:   "10000000",
			SuggestedValue: fmt.Sprintf("%d", int64(10_000_000*1.5)),
			ExpectedImpact: "Increase transaction throughput by ~50% to reduce congestion",
			Confidence:     computeRecommendationConfidence(state.BlockUtilization, 0.8),
			Reasoning: fmt.Sprintf(
				"Block utilization at %.0f%%. Reward: α·ΔPerf(+%.1f) + β·ΔLat(-%.1f) - δ·Stability(%.1f) = net positive",
				state.BlockUtilization*100, pc.alpha*0.5, pc.beta*0.1, pc.delta*0.1,
			),
		}
	}

	// If blocks are consistently <20% full, suggest decreasing gas limit (save resources)
	if state.BlockUtilization < 0.2 && state.BlockHeight > 100 {
		return &types.NetworkRecommendation{
			Parameter:      "max_block_gas",
			CurrentValue:   "10000000",
			SuggestedValue: fmt.Sprintf("%d", int64(10_000_000*0.75)),
			ExpectedImpact: "Reduce node resource usage by ~25% with minimal throughput impact",
			Confidence:     0.7,
			Reasoning: fmt.Sprintf(
				"Block utilization at %.0f%%. Energy savings: γ·ΔEnergy(+%.1f) outweigh performance impact",
				state.BlockUtilization*100, pc.gamma*0.25,
			),
		}
	}

	return nil
}

func (pc *ParameterController) recommendMinCommission(state types.NetworkState) *types.NetworkRecommendation {
	// If validator count is low, suggest lowering commission to attract more
	if state.ActiveValidators < 5 && state.ActiveValidators > 0 {
		return &types.NetworkRecommendation{
			Parameter:      "min_commission_rate",
			CurrentValue:   "0.05",
			SuggestedValue: "0.01",
			ExpectedImpact: "Lower barrier to entry for new validators, improving decentralization",
			Confidence:     0.75,
			Reasoning:      fmt.Sprintf("Only %d active validators; lowering commission floor encourages participation", state.ActiveValidators),
		}
	}

	return nil
}

func (pc *ParameterController) recommendMaxValidators(state types.NetworkState) *types.NetworkRecommendation {
	// If avg block time is fast and network is healthy, consider increasing validator set
	if state.AvgBlockTimeMs > 0 && state.AvgBlockTimeMs < 5000 && state.ActiveValidators >= 10 {
		return &types.NetworkRecommendation{
			Parameter:      "max_validators",
			CurrentValue:   "100",
			SuggestedValue: "125",
			ExpectedImpact: "Allow more validators for improved decentralization",
			Confidence:     0.6,
			Reasoning:      fmt.Sprintf("Network performance healthy (%.0fms avg block time), can support more validators", state.AvgBlockTimeMs),
		}
	}

	return nil
}

func (pc *ParameterController) recommendBlockTime(state types.NetworkState) *types.NetworkRecommendation {
	if state.AvgBlockTimeMs == 0 {
		return nil
	}

	// If block time is consistently slow (>8s), alert
	if state.AvgBlockTimeMs > 8000 {
		return &types.NetworkRecommendation{
			Parameter:      "timeout_commit",
			CurrentValue:   fmt.Sprintf("%.0fms", state.AvgBlockTimeMs),
			SuggestedValue: "5000ms",
			ExpectedImpact: "Reduce block time to improve user experience and throughput",
			Confidence:     0.8,
			Reasoning: fmt.Sprintf(
				"Block time %.0fms exceeds target. Reward: β·ΔLatency(+%.1f) - δ·Stability(%.1f)",
				state.AvgBlockTimeMs, pc.beta*0.4, pc.delta*0.1,
			),
		}
	}

	return nil
}

// computeRecommendationConfidence returns confidence based on how far a metric
// exceeds its threshold.
func computeRecommendationConfidence(value, threshold float64) float64 {
	excess := value - threshold
	if excess <= 0 {
		return 0.5
	}
	// Sigmoid-like mapping: 0.0 excess → 0.6, 0.2 excess → 0.9
	return math.Min(0.6+excess*1.5, 0.95)
}
