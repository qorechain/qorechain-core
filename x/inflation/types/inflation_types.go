package types

import "cosmossdk.io/math"

// EpochInfo tracks the current epoch state.
type EpochInfo struct {
	CurrentEpoch uint64   `json:"current_epoch"`
	CurrentYear  uint64   `json:"current_year"`
	BlockStart   int64    `json:"block_start"`
	TotalMinted  math.Int `json:"total_minted"`
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
