package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
)

// Lock commits part of an existing delegation to its validator until EndTime.
//
// The lock is recorded in delegation SHARES, not tokens. A slash lowers the
// tokens behind every share of the validator, so a lock shrinks exactly with
// the delegation it covers and never claims more than is there. When a
// redelegation slash removes shares from the delegation itself, the keeper
// caps the locks at what the delegation still holds, oldest lock first.
//
// The module keeps its own clock per lock (StartHeight, StartTime, EndTime):
// x/distribution's DelegatorStartingInfo is reset by every reward withdrawal
// and cannot tell how long stake has been committed.
type Lock struct {
	ID        uint64         `json:"id"`
	Delegator string         `json:"delegator"`
	Validator string         `json:"validator"`
	Shares    math.LegacyDec `json:"shares"`
	Months    uint32         `json:"months"`
	// TierBonusBps is the term's bonus when the lock was made. A later change
	// to the tiers does not alter a commitment already made; the light node
	// and validator bonuses and the ceiling are read at every settlement.
	TierBonusBps uint32 `json:"tier_bonus_bps"`
	StartHeight  int64  `json:"start_height"`
	StartTime    int64  `json:"start_time"`
	EndTime      int64  `json:"end_time"`
	// Accrued is bonus credited to this lock and not yet claimed. UnlockEarly
	// forfeits it.
	Accrued math.Int `json:"accrued"`
	// Released is set when the lock expires before its final period was
	// settled: it no longer restricts the delegation and is removed by the
	// next settlement.
	Released bool `json:"released,omitempty"`
}

// Normalized returns the lock with nil amounts read as zero.
func (l Lock) Normalized() Lock {
	if l.Accrued.IsNil() {
		l.Accrued = math.ZeroInt()
	}
	if l.Shares.IsNil() {
		l.Shares = math.LegacyZeroDec()
	}
	return l
}

// Validate checks a lock's own fields.
func (l Lock) Validate() error {
	l = l.Normalized()
	if l.ID == 0 {
		return fmt.Errorf("lock id must be positive")
	}
	if l.Delegator == "" || l.Validator == "" {
		return fmt.Errorf("lock %d: delegator and validator must be set", l.ID)
	}
	if !l.Shares.IsPositive() {
		return fmt.Errorf("lock %d: shares must be positive", l.ID)
	}
	if !allowedMonths(l.Months) {
		return fmt.Errorf("lock %d: months must be one of %v, got %d", l.ID, AllowedLockMonths, l.Months)
	}
	if l.TierBonusBps > BasisPoints {
		return fmt.Errorf("lock %d: tier_bonus_bps must be at most %d", l.ID, BasisPoints)
	}
	if l.StartTime <= 0 || l.EndTime <= l.StartTime {
		return fmt.Errorf("lock %d: end_time must be after a positive start_time", l.ID)
	}
	if l.Accrued.IsNegative() {
		return fmt.Errorf("lock %d: accrued must be non-negative", l.ID)
	}
	return nil
}

// LockEnd is when a lock of months made at start expires: the same day and
// time, months calendar months later, in UTC. Deterministic for a given block
// time.
func LockEnd(start time.Time, months uint32) time.Time {
	return start.UTC().AddDate(0, int(months), 0)
}

// Claimable is a credit owed to an address outside its open locks.
type Claimable struct {
	Address string   `json:"address"`
	Amount  math.Int `json:"amount"`
}
