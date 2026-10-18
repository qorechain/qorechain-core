package types

import "cosmossdk.io/math"

// EpochInfo tracks the current epoch state.
type EpochInfo struct {
	CurrentEpoch uint64   `json:"current_epoch"`
	CurrentYear  uint64   `json:"current_year"`
	BlockStart   int64    `json:"block_start"`
	TotalMinted  math.Int `json:"total_minted"`

	// LastEpochTime is the block time (unix seconds) of the latest epoch
	// boundary, recorded from inflation-target-apy (v3.2.4) so target_apy can
	// size an epoch by the time it really took. Zero, and omitted from the
	// stored JSON, before that.
	LastEpochTime int64 `json:"last_epoch_time,omitempty"`
}

// DefaultEpochInfo returns the initial epoch info.
func DefaultEpochInfo() EpochInfo {
	return EpochInfo{
		CurrentEpoch: 0,
		CurrentYear:  1,
		BlockStart:   0,
		TotalMinted:  math.ZeroInt(),
	}
}
