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
	Address                  string     `json:"address"`
	NodeType                 NodeType   `json:"node_type"`
	Version                  string     `json:"version"`
	Capabilities             []string   `json:"capabilities"`
	Status                   NodeStatus `json:"status"`
	RegisteredAt             int64      `json:"registered_at"`
	LastHeartbeat            int64      `json:"last_heartbeat"`
	TotalHeartbeats          uint64     `json:"total_heartbeats"`
	ExpectedHeartbeats       uint64     `json:"expected_heartbeats"`
	DelegatedStake           string     `json:"delegated_stake"`
	AccumulatedRewards       string     `json:"accumulated_rewards"`

	// InitialHeartbeatInterval records the heartbeat cadence in force when the
	// node registered. It is INFORMATIONAL ONLY: provenance for the operator,
	// surfaced in the query view. It is deliberately NOT used in the uptime
	// maths, because freezing a node's expectation at its registration cadence
	// forever would permanently mis-rate it after any governance change to
	// heartbeat_interval. See ExpectedAccruedThrough for the real mechanism.
	InitialHeartbeatInterval int64 `json:"initial_heartbeat_interval"`

	// ExpectedAccruedThrough is the block height up to which ExpectedHeartbeats
	// has already been accounted for. Expectation is accrued forward from this
	// mark using the interval in force at the time, and never recomputed from
	// the node's whole history, so a governance change to heartbeat_interval
	// only ever affects blocks after the change. Zero means "not yet set"
	// (a record written before this field existed, or a genesis file that omits
	// it); RecordHeartbeat then reconstructs the mark from RegisteredAt and the
	// intervals already charged.
	ExpectedAccruedThrough int64 `json:"expected_accrued_through"`
}

// LightNodeStats tracks aggregate statistics for the lightnode module.
type LightNodeStats struct {
	TotalRegistered  uint64   `json:"total_registered"`
	TotalActive      uint64   `json:"total_active"`
	TotalRewards     math.Int `json:"total_rewards"`
	LastRewardHeight int64    `json:"last_reward_height"`
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
