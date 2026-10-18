package types

import "fmt"

// BasisPoints is 100%.
const BasisPoints = 10_000

// AllowedLockMonths are the lock terms the module knows. Governance sets the
// bonus of each through the lock tiers; a term absent from the tiers is not
// offered.
var AllowedLockMonths = []uint32{3, 6, 12}

// LockTier is one lock term and its yearly bonus in basis points of the locked
// stake.
type LockTier struct {
	Months   uint32 `json:"months"`
	BonusBps uint32 `json:"bonus_bps"`
}

// Params are the stakingtier parameters, stored as JSON. All rates are yearly
// and in basis points; they are net figures, because the module pays the
// bonus straight to the holder and no commission is taken from it.
type Params struct {
	// LockTiers maps a lock term to its bonus: 3 months +1.5, 6 months +3 and
	// 12 months +5 APY points by default.
	LockTiers []LockTier `json:"lock_tiers"`
	// LightNodeBonusBps (+2 points) is added to a lock whose holder runs an
	// active light node: an active lightnode_operator licence and a registered
	// light node with status active and uptime at least the light node
	// module's min_uptime_for_rewards. It applies to locked stake only.
	LightNodeBonusBps uint32 `json:"light_node_bonus_bps"`
	// ValidatorBonusBps (+3 points) is paid on a validator operator's own
	// self-bond, locked or not, while the validator is bonded and not jailed.
	ValidatorBonusBps uint32 `json:"validator_bonus_bps"`
	// MaxTotalAPYBps (25%) caps base yield plus bonus: the bonus of any stake
	// is reduced so that the two together never exceed it.
	MaxTotalAPYBps uint32 `json:"max_total_apy_bps"`
	// EarlyUnlockPenaltyBps (5%) of the locked stake is burned by UnlockEarly.
	EarlyUnlockPenaltyBps uint32 `json:"early_unlock_penalty_bps"`
	// Enabled turns new locks and bonus accrual on or off. Off, existing locks
	// still bind until they expire or are unlocked early, and credited bonus
	// can still be claimed.
	Enabled bool `json:"enabled"`
}

// DefaultParams are the values decided for v3.2.4.
func DefaultParams() Params {
	return Params{
		LockTiers: []LockTier{
			{Months: 3, BonusBps: 150},
			{Months: 6, BonusBps: 300},
			{Months: 12, BonusBps: 500},
		},
		LightNodeBonusBps:     200,
		ValidatorBonusBps:     300,
		MaxTotalAPYBps:        2500,
		EarlyUnlockPenaltyBps: 500,
		Enabled:               true,
	}
}

func allowedMonths(m uint32) bool {
	for _, a := range AllowedLockMonths {
		if a == m {
			return true
		}
	}
	return false
}

// TierBonus returns the bonus of a lock term and whether the term is offered.
func (p Params) TierBonus(months uint32) (uint32, bool) {
	for _, t := range p.LockTiers {
		if t.Months == months {
			return t.BonusBps, true
		}
	}
	return 0, false
}

// Validate checks the parameters.
func (p Params) Validate() error {
	if len(p.LockTiers) == 0 {
		return fmt.Errorf("lock_tiers must offer at least one term")
	}
	for i, t := range p.LockTiers {
		if !allowedMonths(t.Months) {
			return fmt.Errorf("lock_tiers[%d]: months must be one of %v, got %d", i, AllowedLockMonths, t.Months)
		}
		if i > 0 && t.Months <= p.LockTiers[i-1].Months {
			return fmt.Errorf("lock_tiers[%d]: months must be strictly increasing", i)
		}
		if t.BonusBps > BasisPoints {
			return fmt.Errorf("lock_tiers[%d]: bonus_bps must be at most %d, got %d", i, BasisPoints, t.BonusBps)
		}
	}
	for _, f := range []struct {
		name string
		v    uint32
	}{
		{"light_node_bonus_bps", p.LightNodeBonusBps},
		{"validator_bonus_bps", p.ValidatorBonusBps},
		{"early_unlock_penalty_bps", p.EarlyUnlockPenaltyBps},
	} {
		if f.v > BasisPoints {
			return fmt.Errorf("%s must be at most %d, got %d", f.name, BasisPoints, f.v)
		}
	}
	if p.MaxTotalAPYBps == 0 || p.MaxTotalAPYBps > BasisPoints {
		return fmt.Errorf("max_total_apy_bps must be in [1, %d], got %d", BasisPoints, p.MaxTotalAPYBps)
	}
	return nil
}
