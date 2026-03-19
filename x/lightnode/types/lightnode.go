package types

import "cosmossdk.io/math"

// NodeType identifies the category of a light node.
type NodeType string

const (
	NodeTypeSX NodeType = "sx" // state-exchange light node
	NodeTypeUX NodeType = "ux" // user-experience light node
)

// ValidNodeType returns true if the given node type is recognized.
func ValidNodeType(nt NodeType) bool {
	return nt == NodeTypeSX || nt == NodeTypeUX
}

// NodeStatus represents the operational status of a light node.
type NodeStatus string

const (
	NodeStatusActive   NodeStatus = "active"
	NodeStatusInactive NodeStatus = "inactive"
)

// LightNodeInfo describes a registered light node.
type LightNodeInfo struct {
	Address            string     `json:"address"`
	NodeType           NodeType   `json:"node_type"`
	Version            string     `json:"version"`
	Capabilities       []string   `json:"capabilities"`
	Status             NodeStatus `json:"status"`
	RegisteredAt       int64      `json:"registered_at"`
	LastHeartbeat      int64      `json:"last_heartbeat"`
	TotalHeartbeats    uint64     `json:"total_heartbeats"`
	ExpectedHeartbeats uint64     `json:"expected_heartbeats"`
	DelegatedStake     string     `json:"delegated_stake"`
	AccumulatedRewards string     `json:"accumulated_rewards"`
}

// LightNodeStats tracks aggregate statistics for the lightnode module.
type LightNodeStats struct {
	TotalRegistered   uint64   `json:"total_registered"`
	TotalActive       uint64   `json:"total_active"`
	TotalRewards      math.Int `json:"total_rewards"`
	LastRewardHeight  int64    `json:"last_reward_height"`
}

// DefaultLightNodeStats returns zero-valued light node stats.
func DefaultLightNodeStats() LightNodeStats {
	return LightNodeStats{
		TotalRegistered:  0,
		TotalActive:      0,
		TotalRewards:     math.ZeroInt(),
		LastRewardHeight: 0,
	}
}
