package types

// AIConfig holds the AI module configuration.
type AIConfig struct {
	AnomalyThreshold float64 `json:"anomaly_threshold"` // Score above this = anomalous
	RoutingStrategy  string  `json:"routing_strategy"`  // "reputation_weighted" | "round_robin"
	RiskThreshold    float64 `json:"risk_threshold"`    // Score above this = reject contract
	MaxTxPerMinute   int     `json:"max_tx_per_minute"` // Rate limit per sender
}

// DefaultAIConfig returns the default AI module configuration.
func DefaultAIConfig() AIConfig {
	return AIConfig{
		AnomalyThreshold: 0.7,
		RoutingStrategy:  "reputation_weighted",
		RiskThreshold:    0.8,
		MaxTxPerMinute:   10,
	}
}

// AIStats tracks module-level statistics.
type AIStats struct {
	TxsRouted         uint64 `json:"txs_routed"`
	AnomaliesDetected  uint64 `json:"anomalies_detected"`
	ContractsScored    uint64 `json:"contracts_scored"`
	TxsFlagged         uint64 `json:"txs_flagged"`
	TxsRejected        uint64 `json:"txs_rejected"`
}

// FlaggedTx records a flagged transaction.
type FlaggedTx struct {
	TxHash       string   `json:"tx_hash"`
	AnomalyScore float64  `json:"anomaly_score"`
	Flags        []string `json:"flags"`
	Height       int64    `json:"height"`
}
