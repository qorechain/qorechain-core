package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Pool is the module account's accounting and the settlement clock.
//
// The module account holds Outstanding (credited to holders, not yet claimed)
// and a surplus: forfeited bonus and rounding dust. A settlement spends the
// surplus before x/inflation mints anything, so a forfeit lowers future
// emission rather than sitting idle.
type Pool struct {
	Outstanding math.Int `json:"outstanding"`
	// LastSettleTime is the block time (unix seconds) up to which bonus has
	// been settled. Every lock and self-bond accrues from it.
	LastSettleTime   int64 `json:"last_settle_time"`
	LastSettleHeight int64 `json:"last_settle_height"`
	// LastBaseAPYBps is the gross base staking yield of the latest epoch, in
	// basis points; the queries project with it.
	LastBaseAPYBps     math.LegacyDec `json:"last_base_apy_bps"`
	LastOwed           math.Int       `json:"last_owed"`
	LastFunded         math.Int       `json:"last_funded"`
	LastMinted         math.Int       `json:"last_minted"`
	TotalMinted        math.Int       `json:"total_minted"`
	TotalPaid          math.Int       `json:"total_paid"`
	TotalForfeited     math.Int       `json:"total_forfeited"`
	TotalPenaltyBurned math.Int       `json:"total_penalty_burned"`
	// LockCount is the number of open (not released) locks.
	LockCount uint64 `json:"lock_count"`
}

func zeroIfNil(v math.Int) math.Int {
	if v.IsNil() {
		return math.ZeroInt()
	}
	return v
}

// DefaultPool is an empty pool whose clock starts at the first settlement.
func DefaultPool() Pool { return Pool{}.Normalized() }

// Normalized returns the pool with nil amounts read as zero.
func (p Pool) Normalized() Pool {
	p.Outstanding = zeroIfNil(p.Outstanding)
	p.LastOwed = zeroIfNil(p.LastOwed)
	p.LastFunded = zeroIfNil(p.LastFunded)
	p.LastMinted = zeroIfNil(p.LastMinted)
	p.TotalMinted = zeroIfNil(p.TotalMinted)
	p.TotalPaid = zeroIfNil(p.TotalPaid)
	p.TotalForfeited = zeroIfNil(p.TotalForfeited)
	p.TotalPenaltyBurned = zeroIfNil(p.TotalPenaltyBurned)
	if p.LastBaseAPYBps.IsNil() {
		p.LastBaseAPYBps = math.LegacyZeroDec()
	}
	return p
}

// Validate checks that no amount is negative.
func (p Pool) Validate() error {
	p = p.Normalized()
	for name, v := range map[string]math.Int{
		"outstanding": p.Outstanding, "last_owed": p.LastOwed, "last_funded": p.LastFunded,
		"last_minted": p.LastMinted, "total_minted": p.TotalMinted, "total_paid": p.TotalPaid,
		"total_forfeited": p.TotalForfeited, "total_penalty_burned": p.TotalPenaltyBurned,
	} {
		if v.IsNegative() {
			return fmt.Errorf("pool %s must be non-negative", name)
		}
	}
	if p.LastBaseAPYBps.IsNegative() {
		return fmt.Errorf("pool last_base_apy_bps must be non-negative")
	}
	if p.LastSettleTime < 0 || p.LastSettleHeight < 0 {
		return fmt.Errorf("pool settlement clock must be non-negative")
	}
	return nil
}
