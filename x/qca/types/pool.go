package types

// PoolType represents one of the three consensus pools.
type PoolType uint8

const (
	// PoolRPoS is the Reputation Proof-of-Stake pool.
	// Criteria: reputation >= 70th percentile AND stake >= median.
	PoolRPoS PoolType = 0
	// PoolDPoS is the Delegated Proof-of-Stake pool.
	// Criteria: total delegation >= 10,000 QOR.
	PoolDPoS PoolType = 1
	// PoolPoS is the standard Proof-of-Stake pool (remainder).
	PoolPoS PoolType = 2
)

// String returns the human-readable name of a PoolType.
func (p PoolType) String() string {
	switch p {
	case PoolRPoS:
		return "rpos"
	case PoolDPoS:
		return "dpos"
	case PoolPoS:
		return "pos"
	default:
		return "unknown"
	}
}

// PoolClassification records a validator's pool assignment.
type PoolClassification struct {
	ValidatorAddr string   `json:"validator_addr"`
	Pool          PoolType `json:"pool"`
	AssignedAt    int64    `json:"assigned_at"` // block height
}

// PoolConfig holds the triple-pool configuration parameters.
type PoolConfig struct {
	ClassificationInterval uint64 `json:"classification_interval"` // blocks between reclassification (default: 1000)
	WeightRPoS             string `json:"weight_rpos"`             // RPoS pool selection weight (LegacyDec, default: "0.40")
	WeightDPoS             string `json:"weight_dpos"`             // DPoS pool selection weight (LegacyDec, default: "0.35")
	MinDelegationDPoS      uint64 `json:"min_delegation_dpos"`     // minimum delegation for DPoS in uqor (default: 10000000000 = 10k QOR)
	RepPercentileRPoS      uint64 `json:"rep_percentile_rpos"`     // reputation percentile threshold for RPoS (default: 70)
}

// DefaultPoolConfig returns the default pool configuration.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		ClassificationInterval: 1000,
		WeightRPoS:             "0.40",
		WeightDPoS:             "0.35",
		MinDelegationDPoS:      10_000_000_000, // 10k QOR in uqor
		RepPercentileRPoS:      70,
	}
}
