package types

// AIConfig holds the AI module configuration.
type AIConfig struct {
	AnomalyThreshold float64 `json:"anomaly_threshold"` // Score above this = anomalous
	RoutingStrategy  string  `json:"routing_strategy"`  // "reputation_weighted" | "round_robin"
	RiskThreshold    float64 `json:"risk_threshold"`    // Score above this = reject contract
	MaxTxPerMinute   int     `json:"max_tx_per_minute"` // Rate limit per sender

	// The fields below were added in v3.2.2. They are omitted while zero, so
	// a config stored before they existed serialises byte-identically.

	// MaxEVMTxPerMinute is the per-sender limit for EVM-lane transactions;
	// zero means MaxTxPerMinute. The count itself is shared by both lanes.
	MaxEVMTxPerMinute int `json:"max_evm_tx_per_minute,omitempty"`
	// AnomalyAction is AnomalyActionFlag (the default when empty: score,
	// record and emit, never refuse) or AnomalyActionReject (also refuse a
	// transaction scored above AnomalyThreshold).
	AnomalyAction string `json:"anomaly_action,omitempty"`
}

const (
	AnomalyActionFlag   = "flag"
	AnomalyActionReject = "reject"
)

// RejectsAnomalies reports whether an anomaly score may refuse a transaction.
func (c AIConfig) RejectsAnomalies() bool { return c.AnomalyAction == AnomalyActionReject }

// Validate checks the configuration a governance update may store.
func (c AIConfig) Validate() error {
	// Written so that NaN, which compares false with everything, fails too.
	if !(c.AnomalyThreshold >= 0 && c.AnomalyThreshold <= 1) {
		return ErrInvalidConfig.Wrap("anomaly_threshold must be between 0 and 1")
	}
	if !(c.RiskThreshold >= 0 && c.RiskThreshold <= 1) {
		return ErrInvalidConfig.Wrap("risk_threshold must be between 0 and 1")
	}
	if c.RoutingStrategy == "" {
		return ErrInvalidConfig.Wrap("routing_strategy must not be empty")
	}
	if c.MaxTxPerMinute < 0 || c.MaxEVMTxPerMinute < 0 {
		return ErrInvalidConfig.Wrap("rate limits must not be negative")
	}
	switch c.AnomalyAction {
	case "", AnomalyActionFlag, AnomalyActionReject:
	default:
		return ErrInvalidConfig.Wrapf("anomaly_action %q, expected %q or %q", c.AnomalyAction, AnomalyActionFlag, AnomalyActionReject)
	}
	return nil
}

// DefaultAIConfig returns the default AI module configuration.
func DefaultAIConfig() AIConfig {
	return AIConfig{
		AnomalyThreshold: 0.7,
		RoutingStrategy:  "reputation_weighted",
		RiskThreshold:    0.8,
		// The relayer-friendly ceiling the v3.2.0 release handler set on the
		// running networks (app.relayerFriendlyTxPerMinute): about twenty
		// transactions per second from one sender. From v3.2.2 the limit
		// counts every transaction, so a chain started with the old default
		// of 10 would throttle exchanges and faucets at genesis. Existing
		// chains are unaffected: they read the value stored in their state.
		MaxTxPerMinute: 600,
	}
}

// AIStats tracks module-level statistics.
type AIStats struct {
	TxsRouted         uint64 `json:"txs_routed"`
	AnomaliesDetected uint64 `json:"anomalies_detected"`
	ContractsScored   uint64 `json:"contracts_scored"`
	TxsFlagged        uint64 `json:"txs_flagged"`
	TxsRejected       uint64 `json:"txs_rejected"`
}

// FlaggedTx records a flagged transaction.
type FlaggedTx struct {
	TxHash       string   `json:"tx_hash"`
	AnomalyScore float64  `json:"anomaly_score"`
	Flags        []string `json:"flags"`
	Height       int64    `json:"height"`
}
