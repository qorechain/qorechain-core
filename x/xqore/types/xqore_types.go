package types

import (
	"time"

	"cosmossdk.io/math"
)

// XQOREPosition represents a user's locked QORE position.
type XQOREPosition struct {
	Owner      string    `json:"owner"`
	Locked     math.Int  `json:"locked"`      // QORE locked
	XBalance   math.Int  `json:"x_balance"`   // xQORE minted
	LockHeight int64     `json:"lock_height"`
	LockTime   time.Time `json:"lock_time"`
}

// RebaseRecord tracks a PvP rebase event.
type RebaseRecord struct {
	Height        int64    `json:"height"`
	PenaltyAmount math.Int `json:"penalty_amount"`
	BurnedAmount  math.Int `json:"burned_amount"`
	RedistAmount  math.Int `json:"redistributed_amount"`
	TotalXQORE    math.Int `json:"total_xqore_supply"`
}
