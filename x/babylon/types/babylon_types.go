package types

import "time"

// BTCRestakingConfig holds the BTC restaking adapter configuration.
type BTCRestakingConfig struct {
	Enabled           bool   `json:"enabled"`
	MinStakeAmount    int64  `json:"min_stake_amount"`
	UnbondingPeriod   int64  `json:"unbonding_period"`
	CheckpointInterval int64 `json:"checkpoint_interval"`
	BabylonChainID    string `json:"babylon_chain_id"`
}

// DefaultBTCRestakingConfig returns default BTC restaking config.
func DefaultBTCRestakingConfig() BTCRestakingConfig {
	return BTCRestakingConfig{
		Enabled:           false,
		MinStakeAmount:    100000, // 0.001 BTC in satoshis
		UnbondingPeriod:   144,    // ~1 day in BTC blocks
		CheckpointInterval: 10,
		BabylonChainID:    "bbn-1",
	}
}

// BTCStakingPosition represents a BTC staking position.
type BTCStakingPosition struct {
	ID              string    `json:"id"`
	StakerAddress   string    `json:"staker_address"`
	BTCTxHash       string    `json:"btc_tx_hash"`
	AmountSatoshis  int64     `json:"amount_satoshis"`
	StakedAt        time.Time `json:"staked_at"`
	UnbondingHeight int64     `json:"unbonding_height"`
	Status          string    `json:"status"` // active, unbonding, withdrawn
	ValidatorAddr   string    `json:"validator_addr"`
}

// BTCCheckpoint represents a BTC checkpoint record.
type BTCCheckpoint struct {
	EpochNum      uint64    `json:"epoch_num"`
	BTCBlockHash  string    `json:"btc_block_hash"`
	BTCBlockHeight int64    `json:"btc_block_height"`
	StateRoot     string    `json:"state_root"`
	SubmittedAt   time.Time `json:"submitted_at"`
	Status        string    `json:"status"` // pending, confirmed, finalized
}

// BabylonEpochSnapshot captures state at a checkpoint boundary.
type BabylonEpochSnapshot struct {
	EpochNum        uint64 `json:"epoch_num"`
	TotalStaked     int64  `json:"total_staked"`
	ActivePositions int64  `json:"active_positions"`
	ValidatorCount  int64  `json:"validator_count"`
	BlockHeight     int64  `json:"block_height"`
}
