package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// RewardPool divides the lightnode module account balance into the parts the
// distributor may and may not hand out.
//
// The module account receives the light-node slice of the fee split every block
// and pays out claims. Its raw balance is therefore NOT the amount available to
// distribute: it also contains rewards already credited to operators but not yet
// claimed, and whatever accumulated while no node was eligible to receive
// anything. Distributing the raw balance credits both of those again on every
// run, which over-promises rewards the module cannot pay.
//
//	distributable = balance - Reserved - Outstanding
type RewardPool struct {
	// Reserved is the part of the balance held back from proportional
	// distribution. It is set once, to whatever had piled up before the module
	// started accounting for its own liabilities, and moves only by an explicit
	// governance decision. Nobody has earned it, so it is not first-come.
	Reserved math.Int `json:"reserved"`

	// Payable is reserve that governance has explicitly released back into light
	// node distribution. It is exempt from the sweep below, so it drains to
	// operators over time at the capped rate instead of being swept straight back
	// into Reserved on the next settlement.
	Payable math.Int `json:"payable"`

	// Outstanding is the sum of AccumulatedRewards across all registered nodes:
	// rewards credited but not yet claimed. Backed 1:1 by module balance, so it
	// must be excluded from every later distribution.
	Outstanding math.Int `json:"outstanding"`

	// Initialized marks that the split above has been established. A chain that
	// upgraded into this accounting has no record until the coordinated upgrade
	// handler takes the snapshot at the plan height; below that height the
	// EndBlocker writes nothing, so historical replay stays byte-identical.
	Initialized bool `json:"initialized"`

	// LastDistributionHeight is the height of the previous settlement. Reward
	// entitlements accrue per block elapsed since then.
	LastDistributionHeight int64 `json:"last_distribution_height"`

	// LastDistributionTime is the block time (unix seconds) of the previous
	// settlement, written from the lightnode-monthly-cap switch on so the
	// monthly bound can be prorated over wall-clock time. Zero (absent) on every
	// record written before, so those records keep their bytes.
	LastDistributionTime int64 `json:"last_distribution_time,omitempty"`
}

// DefaultRewardPool returns the pool state for a chain starting from genesis:
// nothing accumulated, nothing owed, already accounted for.
func DefaultRewardPool() RewardPool {
	return RewardPool{
		Reserved:    math.ZeroInt(),
		Payable:     math.ZeroInt(),
		Outstanding: math.ZeroInt(),
		Initialized: true,
	}
}

// Normalized returns a copy with nil amounts replaced by zero, so a record
// decoded from a genesis file that omits them is safe to do arithmetic on.
func (p RewardPool) Normalized() RewardPool {
	if p.Reserved.IsNil() {
		p.Reserved = math.ZeroInt()
	}
	if p.Payable.IsNil() {
		p.Payable = math.ZeroInt()
	}
	if p.Outstanding.IsNil() {
		p.Outstanding = math.ZeroInt()
	}
	return p
}

// Validate checks pool correctness.
func (p RewardPool) Validate() error {
	n := p.Normalized()
	if n.Reserved.IsNegative() {
		return fmt.Errorf("reserved must be non-negative, got %s", n.Reserved)
	}
	if n.Payable.IsNegative() {
		return fmt.Errorf("payable must be non-negative, got %s", n.Payable)
	}
	if n.Outstanding.IsNegative() {
		return fmt.Errorf("outstanding must be non-negative, got %s", n.Outstanding)
	}
	return nil
}
