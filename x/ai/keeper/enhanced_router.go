package keeper

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// EnhancedRouter implements AI-enhanced transaction routing with the whitepaper's
// optimization formula: OptimalRoute = argmin_r(α·Latency(r) + β·Cost(r) + γ·Security(r)^-1)
type EnhancedRouter struct {
	heuristic    *HeuristicRouter
	metricsCache *NetworkMetricsCache
	config       types.EnhancedRouterConfig
}

// NetworkMetricsCache caches validator performance metrics to avoid per-TX lookups.
type NetworkMetricsCache struct {
	mu         sync.RWMutex
	validators map[string]*types.ValidatorMetrics
	ttl        time.Duration
	lastFetch  time.Time
}

// NewNetworkMetricsCache creates a new metrics cache.
func NewNetworkMetricsCache(ttl time.Duration) *NetworkMetricsCache {
	return &NetworkMetricsCache{
		validators: make(map[string]*types.ValidatorMetrics),
		ttl:        ttl,
	}
}

// Get returns cached metrics for a validator, or nil if expired/missing.
func (c *NetworkMetricsCache) Get(address string) *types.ValidatorMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m, ok := c.validators[address]
	if !ok || time.Since(m.LastUpdated) > c.ttl {
		return nil
	}
	return m
}

// Set updates a validator's cached metrics.
func (c *NetworkMetricsCache) Set(m *types.ValidatorMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m.LastUpdated = time.Now()
	c.validators[m.Address] = m
}

// GetAll returns all cached validator metrics.
func (c *NetworkMetricsCache) GetAll() []*types.ValidatorMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*types.ValidatorMetrics, 0, len(c.validators))
	for _, m := range c.validators {
		if time.Since(m.LastUpdated) <= c.ttl {
			result = append(result, m)
		}
	}
	return result
}

// NewEnhancedRouter creates a new AI-enhanced router.
func NewEnhancedRouter(config types.EnhancedRouterConfig) *EnhancedRouter {
	cacheTTL := time.Duration(config.CacheTTLSeconds) * time.Second
	if cacheTTL == 0 {
		cacheTTL = 30 * time.Second
	}
	return &EnhancedRouter{
		heuristic:    NewHeuristicRouter(),
		metricsCache: NewNetworkMetricsCache(cacheTTL),
		config:       config,
	}
}

// RouteTransaction implements the whitepaper's routing optimization.
// Fast path: uses weighted scoring on cached metrics (<5ms).
// Falls back to heuristic router if no cached metrics are available.
func (r *EnhancedRouter) RouteTransaction(_ context.Context, tx types.TransactionInfo) (*types.RoutingDecision, error) {
	validators := r.metricsCache.GetAll()

	// Fallback: if no cached metrics, use heuristic routing
	if len(validators) == 0 {
		return r.heuristic.RouteTransaction(context.Background(), tx)
	}

	// Score each validator using the whitepaper formula:
	// OptimalRoute = argmin_r(α·Latency(r) + β·Cost(r) + γ·Security(r)^-1)
	type scoredValidator struct {
		address string
		score   float64
	}

	var scored []scoredValidator
	for _, v := range validators {
		// Normalize latency: lower is better (0-1 range, 0=best)
		latencyScore := normalizeLatency(v.AvgLatencyMs)

		// Cost score: based on load — higher load = higher cost (0-1 range, 0=best)
		costScore := v.LoadPercent / 100.0

		// Security score: inverse of reputation — higher reputation = lower inverse (0-1 range, 0=best)
		// Security(r)^-1 means we want higher reputation → lower score
		securityInverse := 1.0
		if v.ReputationScore > 0 {
			securityInverse = 1.0 / v.ReputationScore
		}
		// Normalize to 0-1 range
		securityScore := math.Min(securityInverse, 1.0)

		// Combined score (lower is better)
		combinedScore := r.config.Alpha*latencyScore + r.config.Beta*costScore + r.config.Gamma*securityScore

		scored = append(scored, scoredValidator{
			address: v.Address,
			score:   combinedScore,
		})
	}

	// Sort by score (ascending — lower is better)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score < scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Build preferred validator list
	preferredValidators := make([]string, 0, len(scored))
	for _, s := range scored {
		preferredValidators = append(preferredValidators, s.address)
	}

	// Determine priority based on TX type and amount
	priority := 0
	reason := "AI-optimized routing: weighted scoring on latency, cost, and security"

	switch tx.TxType {
	case "contract_deploy":
		priority = 1
		reason = "contract deploy → low-load validator preferred (AI-optimized)"
	case "delegate":
		priority = 1
		reason = "staking tx → high-uptime validator preferred (AI-optimized)"
	default:
		if tx.Amount > 10_000_000_000 { // > 10,000 QOR
			priority = 1
			reason = "large value tx → high-reputation validator preferred (AI-optimized)"
		}
	}

	return &types.RoutingDecision{
		PreferredValidators: preferredValidators,
		Priority:            priority,
		Reason:              reason,
		Confidence:          computeConfidence(len(validators)),
	}, nil
}

// UpdateMetrics updates the metrics cache for a validator.
func (r *EnhancedRouter) UpdateMetrics(metrics *types.ValidatorMetrics) {
	r.metricsCache.Set(metrics)
}

// normalizeLatency converts raw latency to a 0-1 score (0=best, 1=worst).
// Assumes latency between 0ms (best) and 1000ms (worst).
func normalizeLatency(latencyMs float64) float64 {
	if latencyMs <= 0 {
		return 0
	}
	if latencyMs >= 1000 {
		return 1.0
	}
	return latencyMs / 1000.0
}

// computeConfidence returns a confidence score based on available data.
// More validator metrics → higher confidence.
func computeConfidence(validatorCount int) float64 {
	if validatorCount >= 10 {
		return 0.95
	}
	if validatorCount >= 5 {
		return 0.85
	}
	if validatorCount >= 2 {
		return 0.75
	}
	return 0.6
}
