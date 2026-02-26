package types

import "fmt"

// Params holds module-level parameters for the RDK module.
type Params struct {
	MaxRollups             uint32 `json:"max_rollups"`
	MinStakeForRollup      int64  `json:"min_stake_for_rollup"`      // uqor
	RollupCreationBurnRate string `json:"rollup_creation_burn_rate"` // decimal (e.g. "0.01" = 1%)
	DefaultChallengeWindow uint64 `json:"default_challenge_window"`  // seconds
	MaxDABlobSize          uint64 `json:"max_da_blob_size"`          // bytes
	BlobRetentionBlocks    uint64 `json:"blob_retention_blocks"`
	MaxBatchesPerBlock     uint32 `json:"max_batches_per_block"`
}

// DefaultParams returns sensible default parameters.
func DefaultParams() Params {
	return Params{
		MaxRollups:             100,
		MinStakeForRollup:      10000000000, // 10,000 QOR in uqor
		RollupCreationBurnRate: "0.01",      // 1%
		DefaultChallengeWindow: 604800,      // 7 days in seconds
		MaxDABlobSize:          2097152,     // 2 MB
		BlobRetentionBlocks:    432000,      // ~30 days at 6s blocks
		MaxBatchesPerBlock:     10,
	}
}

// GenesisState defines the rdk module genesis state.
type GenesisState struct {
	Params  Params            `json:"params"`
	Rollups []RollupConfig    `json:"rollups"`
	Batches []SettlementBatch `json:"batches"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Rollups: []RollupConfig{},
		Batches: []SettlementBatch{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params.MaxRollups == 0 {
		return fmt.Errorf("max rollups must be positive")
	}
	if gs.Params.MinStakeForRollup <= 0 {
		return fmt.Errorf("min stake for rollup must be positive")
	}
	if gs.Params.MaxDABlobSize == 0 {
		return fmt.Errorf("max DA blob size must be positive")
	}
	if gs.Params.MaxBatchesPerBlock == 0 {
		return fmt.Errorf("max batches per block must be positive")
	}
	for i, r := range gs.Rollups {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("rollup %d: %w", i, err)
		}
	}
	return nil
}
