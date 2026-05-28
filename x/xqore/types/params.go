package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
)

// PenaltyTier defines an exit penalty for a time bracket.
type PenaltyTier struct {
	MinDuration time.Duration  `json:"min_duration"`
	PenaltyRate math.LegacyDec `json:"penalty_rate"`
}

// LockMultiplierTier defines a voting power multiplier for a lock duration bracket.
type LockMultiplierTier struct {
	MinDuration time.Duration  `json:"min_duration"`
	Multiplier  math.LegacyDec `json:"multiplier"`
}

// Params defines the configurable parameters for the xQORE module.
type Params struct {
	GovernanceMultiplier   math.LegacyDec       `json:"governance_multiplier"` // max voting power multiplier (2.0x)
	MinLockAmount          math.Int             `json:"min_lock_amount"`       // minimum QORE to lock
	ExitPenaltySchedule    []PenaltyTier        `json:"exit_penalty_schedule"`
	LockMultiplierSchedule []LockMultiplierTier `json:"lock_multiplier_schedule"` // duration-based VP multipliers
	PenaltyBurnRate        math.LegacyDec       `json:"penalty_burn_rate"`        // 0 = 100% redistributed (PvP)
	RebaseInterval         int64                `json:"rebase_interval"`          // blocks between rebases
	Enabled                bool                 `json:"enabled"`
}

// DefaultParams returns the default xQORE parameters.
func DefaultParams() Params {
	return Params{
		GovernanceMultiplier: math.LegacyNewDec(2),
		MinLockAmount:        math.NewInt(1_000_000), // 1 QOR in uqor
		ExitPenaltySchedule: []PenaltyTier{
			{MinDuration: 0, PenaltyRate: math.LegacyNewDecWithPrec(50, 2)},                   // immediate: 50%
			{MinDuration: 30 * 24 * time.Hour, PenaltyRate: math.LegacyNewDecWithPrec(35, 2)}, // 1 month: 35%
			{MinDuration: 90 * 24 * time.Hour, PenaltyRate: math.LegacyNewDecWithPrec(15, 2)}, // 3 months: 15%
			{MinDuration: 180 * 24 * time.Hour, PenaltyRate: math.LegacyZeroDec()},            // 6 months: 0%
		},
		LockMultiplierSchedule: []LockMultiplierTier{
			{MinDuration: 0, Multiplier: math.LegacyOneDec()},                                    // no lock: 1.0x
			{MinDuration: 90 * 24 * time.Hour, Multiplier: math.LegacyMustNewDecFromStr("1.2")},  // 3 months: 1.2x
			{MinDuration: 180 * 24 * time.Hour, Multiplier: math.LegacyMustNewDecFromStr("1.4")}, // 6 months: 1.4x
			{MinDuration: 365 * 24 * time.Hour, Multiplier: math.LegacyMustNewDecFromStr("1.7")}, // 12 months: 1.7x
			{MinDuration: 2 * 365 * 24 * time.Hour, Multiplier: math.LegacyNewDec(2)},            // 24 months: 2.0x
		},
		PenaltyBurnRate: math.LegacyZeroDec(), // 0% burned — 100% redistributed (PvP rebase per QCW2)
		RebaseInterval:  100,                  // every 100 blocks
		Enabled:         true,
	}
}

// Validate checks param correctness.
func (p Params) Validate() error {
	if p.GovernanceMultiplier.IsNegative() {
		return fmt.Errorf("governance_multiplier must be non-negative")
	}
	if p.MinLockAmount.IsNegative() {
		return fmt.Errorf("min_lock_amount must be non-negative")
	}
	if len(p.ExitPenaltySchedule) == 0 {
		return fmt.Errorf("exit_penalty_schedule must not be empty")
	}
	for i, tier := range p.ExitPenaltySchedule {
		if tier.PenaltyRate.IsNegative() || tier.PenaltyRate.GT(math.LegacyOneDec()) {
			return fmt.Errorf("penalty_schedule[%d]: rate must be between 0 and 1", i)
		}
	}
	// Verify penalty tiers are sorted ascending by MinDuration
	for i := 1; i < len(p.ExitPenaltySchedule); i++ {
		if p.ExitPenaltySchedule[i].MinDuration <= p.ExitPenaltySchedule[i-1].MinDuration {
			return fmt.Errorf("penalty_schedule[%d]: min_duration must be strictly increasing", i)
		}
	}
	// Verify lock multiplier tiers
	for i, tier := range p.LockMultiplierSchedule {
		if tier.Multiplier.IsNegative() {
			return fmt.Errorf("lock_multiplier_schedule[%d]: multiplier must be non-negative", i)
		}
	}
	for i := 1; i < len(p.LockMultiplierSchedule); i++ {
		if p.LockMultiplierSchedule[i].MinDuration <= p.LockMultiplierSchedule[i-1].MinDuration {
			return fmt.Errorf("lock_multiplier_schedule[%d]: min_duration must be strictly increasing", i)
		}
	}
	if p.PenaltyBurnRate.IsNegative() || p.PenaltyBurnRate.GT(math.LegacyOneDec()) {
		return fmt.Errorf("penalty_burn_rate must be between 0 and 1")
	}
	if p.RebaseInterval < 1 {
		return fmt.Errorf("rebase_interval must be >= 1")
	}
	return nil
}
