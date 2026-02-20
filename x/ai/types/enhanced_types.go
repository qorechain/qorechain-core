package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---- Enhanced Router Types ----

// EnhancedRouterConfig holds configurable weights for the AI-enhanced router.
type EnhancedRouterConfig struct {
	Alpha           float64 `json:"alpha"`              // Latency weight (default 0.4)
	Beta            float64 `json:"beta"`               // Cost weight (default 0.3)
	Gamma           float64 `json:"gamma"`              // Security weight (default 0.3)
	UseSidecar      bool    `json:"use_sidecar"`        // Enable QCAI Backend-enhanced routing
	CacheTTLSeconds int     `json:"cache_ttl_seconds"`  // Metrics cache TTL
}

// DefaultEnhancedRouterConfig returns default enhanced router weights.
func DefaultEnhancedRouterConfig() EnhancedRouterConfig {
	return EnhancedRouterConfig{
		Alpha:           0.4,
		Beta:            0.3,
		Gamma:           0.3,
		UseSidecar:      false,
		CacheTTLSeconds: 30,
	}
}

// ValidatorMetrics holds cached performance data for a validator.
type ValidatorMetrics struct {
	Address        string  `json:"address"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	UptimePercent  float64 `json:"uptime_percent"`
	LoadPercent    float64 `json:"load_percent"`
	ReputationScore float64 `json:"reputation_score"`
	LastUpdated    time.Time `json:"last_updated"`
}

// ---- Fraud Detection Types ----

// FraudResult contains the output of fraud detection analysis.
type FraudResult struct {
	ThreatLevel     string  `json:"threat_level"`      // "none" | "low" | "medium" | "high" | "critical"
	ThreatType      string  `json:"threat_type"`       // "sybil" | "ddos" | "flash_loan" | "exploit" | "unknown"
	Action          string  `json:"action"`            // "allow" | "rate_limit" | "circuit_break" | "alert"
	Confidence      float64 `json:"confidence"`
	Details         string  `json:"details"`
	InvestigationID string  `json:"investigation_id"`  // Non-empty if action != "allow"
}

// FraudInvestigation stores details of a fraud investigation.
type FraudInvestigation struct {
	ID          string    `json:"id"`
	ThreatType  string    `json:"threat_type"`
	ThreatLevel string    `json:"threat_level"`
	Sender      string    `json:"sender"`
	Details     string    `json:"details"`
	TxHash      string    `json:"tx_hash"`
	Height      int64     `json:"height"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

// ---- Fee Optimization Types ----

// FeeEstimate represents a predicted fee for a given urgency level.
type FeeEstimate struct {
	SuggestedFee        sdk.Coin `json:"suggested_fee"`
	EstimatedBlocks     int      `json:"estimated_blocks"`
	CurrentCongestion   float64  `json:"current_congestion"`    // 0.0 (empty) to 1.0 (full)
	PredictedCongestion float64  `json:"predicted_congestion"`  // Next 10 blocks prediction
	Confidence          float64  `json:"confidence"`
}

// FeeSnapshot records fee state at a given block height.
type FeeSnapshot struct {
	Height              int64   `json:"height"`
	AvgFee              uint64  `json:"avg_fee"`
	Congestion          float64 `json:"congestion"`
	PredictedCongestion float64 `json:"predicted_congestion"`
}

// ---- Network Optimization Types ----

// NetworkRecommendation suggests a parameter adjustment.
type NetworkRecommendation struct {
	Parameter      string  `json:"parameter"`       // e.g., "max_block_gas", "min_commission_rate"
	CurrentValue   string  `json:"current_value"`
	SuggestedValue string  `json:"suggested_value"`
	ExpectedImpact string  `json:"expected_impact"`
	Confidence     float64 `json:"confidence"`
	Reasoning      string  `json:"reasoning"`
}

// NetworkState holds the current network performance metrics.
type NetworkState struct {
	BlockHeight      int64   `json:"block_height"`
	AvgBlockTimeMs   float64 `json:"avg_block_time_ms"`
	TxThroughput     float64 `json:"tx_throughput"`       // TX per second
	PendingTxCount   int     `json:"pending_tx_count"`
	ActiveValidators int     `json:"active_validators"`
	BlockUtilization float64 `json:"block_utilization"`   // 0.0 to 1.0
}

// ---- Circuit Breaker Types ----

// CircuitBreakerState tracks circuit breaker status for a contract.
type CircuitBreakerState struct {
	ContractAddr string    `json:"contract_addr"`
	Reason       string    `json:"reason"`
	ActivatedAt  time.Time `json:"activated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// ---- Extended Stats ----

// ExtendedAIStats adds Phase 2 counters to the base AIStats.
type ExtendedAIStats struct {
	AIStats

	FraudDetections    uint64 `json:"fraud_detections"`
	FeeEstimates       uint64 `json:"fee_estimates"`
	NetworkOptRuns     uint64 `json:"network_opt_runs"`
	CircuitBreaks      uint64 `json:"circuit_breaks"`
	ContractsGenerated uint64 `json:"contracts_generated"`
	ContractsAudited   uint64 `json:"contracts_audited"`
}
