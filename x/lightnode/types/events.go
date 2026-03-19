package types

// Event types for the lightnode module.
const (
	EventTypeLightNodeRegister           = "lightnode_register"
	EventTypeLightNodeDeregister         = "lightnode_deregister"
	EventTypeLightNodeHeartbeat          = "lightnode_heartbeat"
	EventTypeLightNodeClaimRewards       = "lightnode_claim_rewards"
	EventTypeLightNodeStatusChange       = "lightnode_status_change"
	EventTypeLightNodeRewardDistribution = "lightnode_reward_distribution"

	AttributeKeyAddress      = "address"
	AttributeKeyNodeType     = "node_type"
	AttributeKeyVersion      = "version"
	AttributeKeyStatus       = "status"
	AttributeKeyRewardAmount = "reward_amount"
	AttributeKeyBlockHeight  = "block_height"
)
