package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Params defines the configurable parameters for the lightnode module.
type Params struct {
	RegistrationFee     math.Int       `json:"registration_fee"`      // uqor fee to register a light node
	HeartbeatInterval   int64          `json:"heartbeat_interval"`    // blocks between required heartbeats
	MinDelegatedStake   math.Int       `json:"min_delegated_stake"`   // minimum uqor delegated to be eligible
	RewardShare         math.LegacyDec `json:"reward_share"`          // fraction of block rewards for light nodes
	MinUptimeForRewards math.LegacyDec `json:"min_uptime_for_rewards"` // minimum uptime ratio to qualify for rewards
	MaxLightNodes       uint64         `json:"max_light_nodes"`       // maximum number of registered light nodes
	HeartbeatGracePeriod int64         `json:"heartbeat_grace_period"` // extra blocks allowed before marking inactive
}

// DefaultParams returns the default lightnode module parameters.
func DefaultParams() Params {
	return Params{
		RegistrationFee:      math.NewInt(1_000_000),              // 1 QOR in uqor
		HeartbeatInterval:    1000,                                 // every 1000 blocks
		MinDelegatedStake:    math.NewInt(100_000_000),            // 100 QOR in uqor
		RewardShare:          math.LegacyNewDecWithPrec(3, 2),     // 0.03 (3%)
		MinUptimeForRewards:  math.LegacyNewDecWithPrec(80, 2),   // 0.80 (80%)
		MaxLightNodes:        10000,
		HeartbeatGracePeriod: 100,
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
	return nil
}
