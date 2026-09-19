package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StrandedRewardsKey stores the record below.
var StrandedRewardsKey = []byte("burn/stranded")

// StrandedRewards records the staker share this module deposited into the
// distribution module's BALANCE without recording it as anyone's outstanding
// reward.
//
// Those coins back no claim. The distribution module pays delegators from
// ValidatorOutstandingRewards, and a plain module-to-module transfer never
// touches that ledger, so every emission added coins nobody could withdraw.
// The leak is closed from the release height; this record is what the already
// stranded amount becomes, and it moves only by a governance vote, exactly
// like the light-node reserve.
type StrandedRewards struct {
	// Reserved is what is left to release, in uqor.
	Reserved math.Int `json:"reserved"`
	// Released is how much governance has already moved out, in uqor.
	Released math.Int `json:"released"`
	// Initialized marks that the snapshot has been taken. Below the release
	// height nothing is written, so history replays byte-identically.
	Initialized bool `json:"initialized"`
	// SnapshotHeight is the height the snapshot was taken at.
	SnapshotHeight int64 `json:"snapshot_height"`
}

// Normalized replaces nil amounts with zero.
func (s StrandedRewards) Normalized() StrandedRewards {
	if s.Reserved.IsNil() {
		s.Reserved = math.ZeroInt()
	}
	if s.Released.IsNil() {
		s.Released = math.ZeroInt()
	}
	return s
}

// Validate checks the record.
func (s StrandedRewards) Validate() error {
	n := s.Normalized()
	if n.Reserved.IsNegative() {
		return fmt.Errorf("reserved must be non-negative, got %s", n.Reserved)
	}
	if n.Released.IsNegative() {
		return fmt.Errorf("released must be non-negative, got %s", n.Released)
	}
	return nil
}

// ValidateBasic checks the release message without reading state.
func (msg MsgReleaseStrandedRewards) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Destination); err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}
	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return fmt.Errorf("amount %q is not an integer number of uqor", msg.Amount)
	}
	if !amount.IsPositive() {
		return fmt.Errorf("amount must be positive, got %s", msg.Amount)
	}
	return nil
}
