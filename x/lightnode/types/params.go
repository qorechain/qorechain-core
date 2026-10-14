package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Params defines the configurable parameters for the lightnode module.
type Params struct {
	RegistrationFee      math.Int       `json:"registration_fee"`       // uqor fee to register a light node
	HeartbeatInterval    int64          `json:"heartbeat_interval"`     // blocks between required heartbeats
	MinDelegatedStake    math.Int       `json:"min_delegated_stake"`    // minimum uqor delegated to be eligible
	RewardShare          math.LegacyDec `json:"reward_share"`           // fraction of block rewards for light nodes
	MinUptimeForRewards  math.LegacyDec `json:"min_uptime_for_rewards"` // minimum uptime ratio to qualify for rewards
	MaxLightNodes        uint64         `json:"max_light_nodes"`        // maximum number of registered light nodes
	HeartbeatGracePeriod int64          `json:"heartbeat_grace_period"` // extra blocks allowed before marking inactive

	// MaxRewardPerBlockPerNode bounds what one active light node can be credited
	// per block, in uqor. It exists because the pot and the participation are set
	// independently: the fee split pays the module a fixed slice of each epoch's
	// emission whatever the participation, so a purely proportional payout hands a
	// single operator the entire slice.
	//
	// It is a BOUND on the proportional split, not a replacement for it. A node
	// receives min(its proportional share, this bound x blocks elapsed x uptime).
	// With few operators everyone sits at the bound and earns a predictable,
	// sellable amount; once enough operators join that the proportional share
	// falls below the bound, the original stake-weighted distribution takes over
	// again on its own. The pathological case is fenced off without changing the
	// product.
	//
	// Sized from the operator's actual outlay, which is the 50 USD licence rather
	// than the 1,000 QOR bond: 2,000 QOR/month per active node. At 2.22 s/block
	// that is 2,592,000/2.22 = 1,167,567 blocks/month, so 1,713 uqor/block gives
	// 2,000.04 QOR/month. (Measured cadence is 2.322 s/block, at which the same
	// constant yields ~1,912 QOR/month; 1,790 would hit 2,000 exactly.)
	//
	// At the 0.00517 USD/QOR sale-round reference that is 10.34 USD/month plus
	// ~2.09 USD of staking, recovering a 50 USD licence in about four months.
	// The 16 current buyers cost 32,000 QOR/month out of a 2.23M QOR fund - 1.4%
	// - and the fund carries over 1,100 operators before it needs revisiting.
	// Uncapped, the first registrant takes the whole fund and recovers the licence
	// in 187 minutes.
	//
	// Governance can adjust it, but asymmetrically: raising it is easy, lowering
	// it after operators have been paid reads as a withdrawn promise. Hence a
	// conservative value rather than the most the fund would bear.
	//
	// A POINTER, deliberately. Params are stored as JSON, and math.Int is a
	// struct, which encoding/json never treats as empty: a value type here would
	// serialise "0" for a record that never had the field, so the bytes the
	// new binary writes at height 0 (and on every SetParams) would differ from
	// the ones the network's history holds, and a replay from genesis would
	// diverge. With a pointer an absent field stays absent, byte for byte.
	// Read it through MaxRewardPerBlock(), never directly.
	MaxRewardPerBlockPerNode *math.Int `json:"max_reward_per_block_per_node,omitempty"`

	// MaxRewardPerMonthPerNode is the same bound stated the way it was decided:
	// 2,000 QOR per 30 days per active node, prorated over the wall-clock time
	// since the previous settlement (block time, which every node agrees on).
	// The per-block form only pays 2,000 at the cadence it was derived at
	// (2.22 s); at the 3.2 s mainnet ran in October it paid ~1,381. From the
	// lightnode-monthly-cap switch this bound replaces the per-block one.
	// A pointer for the same byte-identity reason as the field above; read it
	// through MaxRewardPerMonth().
	MaxRewardPerMonthPerNode *math.Int `json:"max_reward_per_month_per_node,omitempty"`
}

// defaultMaxRewardPerBlockPerNode is the bound a record that predates the field
// is read with: 1,713 uqor per block, about 2,000 QOR per month per active node.
var defaultMaxRewardPerBlockPerNode = math.NewInt(1713)

// MaxRewardPerBlock is the per-node bound to use, with the default standing in
// for a record written before the field existed. It does not materialise the
// field on the receiver, so a later SetParams still writes the old shape.
func (p Params) MaxRewardPerBlock() math.Int {
	if p.MaxRewardPerBlockPerNode == nil || p.MaxRewardPerBlockPerNode.IsNil() {
		return defaultMaxRewardPerBlockPerNode
	}
	return *p.MaxRewardPerBlockPerNode
}

// defaultMaxRewardPerMonthPerNode is the decided 2,000 QOR per month per active
// node (2026-08-24), read for any record that predates the field.
var defaultMaxRewardPerMonthPerNode = math.NewInt(2_000_000_000)

// SecondsPerRewardMonth is the 30-day month the monthly bound is stated in.
const SecondsPerRewardMonth int64 = 2_592_000

// MaxRewardPerMonth is the monthly per-node bound to use, with the default
// standing in for a record written before the field existed.
func (p Params) MaxRewardPerMonth() math.Int {
	if p.MaxRewardPerMonthPerNode == nil || p.MaxRewardPerMonthPerNode.IsNil() {
		return defaultMaxRewardPerMonthPerNode
	}
	return *p.MaxRewardPerMonthPerNode
}

// DefaultParams returns the default lightnode module parameters.
func DefaultParams() Params {
	return Params{
		RegistrationFee:          math.NewInt(1_000_000),           // 1 QOR in uqor
		HeartbeatInterval:        1000,                             // every 1000 blocks
		MinDelegatedStake:        math.NewInt(1_000_000_000),       // 1000 QOR in uqor
		RewardShare:              math.LegacyNewDecWithPrec(3, 2),  // 0.03 (3%)
		MinUptimeForRewards:      math.LegacyNewDecWithPrec(80, 2), // 0.80 (80%)
		MaxLightNodes:            10000,
		HeartbeatGracePeriod:     100,
		MaxRewardPerBlockPerNode: &defaultMaxRewardPerBlockPerNode, // ~2,000 QOR/month per active node
	}
}

// Validate checks param correctness.
func (p Params) Validate() error {
	if p.RegistrationFee.IsNegative() {
		return fmt.Errorf("registration_fee must be non-negative, got %s", p.RegistrationFee)
	}
	if p.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat_interval must be positive, got %d", p.HeartbeatInterval)
	}
	if p.MinDelegatedStake.IsNegative() {
		return fmt.Errorf("min_delegated_stake must be non-negative, got %s", p.MinDelegatedStake)
	}
	if p.RewardShare.IsNegative() || p.RewardShare.GT(math.LegacyOneDec()) {
		return fmt.Errorf("reward_share must be between 0 and 1, got %s", p.RewardShare)
	}
	if p.MinUptimeForRewards.IsNegative() || p.MinUptimeForRewards.GT(math.LegacyOneDec()) {
		return fmt.Errorf("min_uptime_for_rewards must be between 0 and 1, got %s", p.MinUptimeForRewards)
	}
	if p.MaxLightNodes == 0 {
		return fmt.Errorf("max_light_nodes must be positive")
	}
	if p.HeartbeatGracePeriod < 0 {
		return fmt.Errorf("heartbeat_grace_period must be non-negative, got %d", p.HeartbeatGracePeriod)
	}
	// Absent is valid: every record written before the field existed, and
	// the genuine genesis files of both networks, lack it. Refusing it here
	// is what made a previous cut reject the real mainnet genesis and fall at
	// height 0 instead of starting.
	if p.MaxRewardPerBlockPerNode != nil && !p.MaxRewardPerBlockPerNode.IsNil() &&
		p.MaxRewardPerBlockPerNode.IsNegative() {
		return fmt.Errorf("max_reward_per_block_per_node must be non-negative, got %s", p.MaxRewardPerBlockPerNode)
	}
	if p.MaxRewardPerMonthPerNode != nil && !p.MaxRewardPerMonthPerNode.IsNil() &&
		p.MaxRewardPerMonthPerNode.IsNegative() {
		return fmt.Errorf("max_reward_per_month_per_node must be non-negative, got %s", p.MaxRewardPerMonthPerNode)
	}
	return nil
}
