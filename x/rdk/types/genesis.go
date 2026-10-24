package types

import (
	"fmt"

	"cosmossdk.io/math"
)

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

// Bounds on a governance parameter update (ValidateUpdate). They keep a
// proposal from switching the module off by accident; they are not policy.
const (
	// MinChallengeWindowSec is the shortest default challenge window: one
	// minute, so a watcher always has at least a few blocks to challenge.
	MinChallengeWindowSec uint64 = 60
	// MaxChallengeWindowSec is the longest: 30 days.
	MaxChallengeWindowSec uint64 = 30 * 24 * 3600
	// MaxDABlobSizeLimit caps max_da_blob_size at 8 MiB, under any block size
	// the chain runs with.
	MaxDABlobSizeLimit uint64 = 8 << 20
)

// ValidateUpdate checks a parameter set proposed through MsgUpdateParams
// (from v3.2.5, FeatureRDKParamsRoute). It is stricter than the genesis check,
// which existing genesis files must keep passing.
func (p Params) ValidateUpdate() error {
	if p.MaxRollups == 0 {
		return ErrInvalidParams.Wrap("max_rollups must be positive")
	}
	if p.MinStakeForRollup <= 0 {
		return ErrInvalidParams.Wrap("min_stake_for_rollup must be positive")
	}
	rate, err := math.LegacyNewDecFromStr(p.RollupCreationBurnRate)
	if err != nil || rate.IsNegative() || rate.GTE(math.LegacyOneDec()) {
		return ErrInvalidParams.Wrapf("rollup_creation_burn_rate must be a decimal in [0, 1), got %q", p.RollupCreationBurnRate)
	}
	if p.DefaultChallengeWindow < MinChallengeWindowSec || p.DefaultChallengeWindow > MaxChallengeWindowSec {
		return ErrInvalidParams.Wrapf("default_challenge_window must be %d..%d seconds, got %d",
			MinChallengeWindowSec, MaxChallengeWindowSec, p.DefaultChallengeWindow)
	}
	if p.MaxDABlobSize == 0 || p.MaxDABlobSize > MaxDABlobSizeLimit {
		return ErrInvalidParams.Wrapf("max_da_blob_size must be 1..%d bytes, got %d", MaxDABlobSizeLimit, p.MaxDABlobSize)
	}
	if p.BlobRetentionBlocks == 0 {
		return ErrInvalidParams.Wrap("blob_retention_blocks must be positive")
	}
	if p.MaxBatchesPerBlock == 0 {
		return ErrInvalidParams.Wrap("max_batches_per_block must be positive")
	}
	return nil
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
