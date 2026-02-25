package types

// Reward records the computed reward signal at a specific block height.
// All numeric fields are stored as LegacyDec string representations.
type Reward struct {
	Height               int64  `json:"height"`
	TotalReward          string `json:"total_reward"`
	ThroughputDelta      string `json:"throughput_delta"`
	FinalityDelta        string `json:"finality_delta"`
	DecentralizationDelta string `json:"decentralization_delta"`
	MEVEstimate          string `json:"mev_estimate"`
	FailedTxRatio        string `json:"failed_tx_ratio"`
}
