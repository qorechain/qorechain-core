package types

import "context"

// AIEngine is the interface for AI services.
// MVP: implemented by heuristics. Future: implemented by ML models.
type AIEngine interface {
	RouteTransaction(ctx context.Context, tx TransactionInfo) (*RoutingDecision, error)
	DetectAnomaly(ctx context.Context, tx TransactionInfo, history []TransactionInfo) (*AnomalyResult, error)
	ScoreContractRisk(ctx context.Context, code []byte, chain string) (*RiskScore, error)
}

// TransactionInfo contains the data needed for AI analysis.
type TransactionInfo struct {
	TxHash   string `json:"tx_hash"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Amount   uint64 `json:"amount"`
	TxType   string `json:"tx_type"` // "transfer" | "delegate" | "contract_deploy" | "contract_call"
	GasUsed  uint64 `json:"gas_used"`
	Height   int64  `json:"height"`
}

// RoutingDecision is the output of the routing engine.
type RoutingDecision struct {
	PreferredValidators []string `json:"preferred_validators"`
	Priority            int      `json:"priority"` // 0=normal, 1=high, 2=critical
	Reason              string   `json:"reason"`
	Confidence          float64  `json:"confidence"`
}

// AnomalyResult is the output of anomaly detection.
type AnomalyResult struct {
	IsAnomalous bool     `json:"is_anomalous"`
	Score       float64  `json:"score"`      // 0.0 (normal) to 1.0 (highly anomalous)
	Flags       []string `json:"flags"`      // e.g., ["high_frequency", "unusual_amount"]
	Action      string   `json:"action"`     // "allow" | "flag" | "reject"
	Confidence  float64  `json:"confidence"`
}

// RiskScore is the output of contract risk scoring.
type RiskScore struct {
	Score          float64     `json:"score"`    // 0.0 (safe) to 1.0 (critical risk)
	Severity       string      `json:"severity"` // "LOW" | "MEDIUM" | "HIGH" | "CRITICAL"
	Issues         []RiskIssue `json:"issues"`
	Recommendation string      `json:"recommendation"` // "deploy" | "review" | "reject"
}

// RiskIssue describes a specific security issue found.
type RiskIssue struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}
