package types

const (
	EventTypeObservationCollected    = "rl_observation_collected"
	EventTypeActionApplied           = "rl_action_applied"
	EventTypeCircuitBreakerTriggered = "rl_circuit_breaker_triggered"
	EventTypeCircuitBreakerRecovered = "rl_circuit_breaker_recovered"
	EventTypeAgentModeChanged        = "rl_agent_mode_changed"
	EventTypePolicyUpdated           = "rl_policy_updated"
	EventTypeRewardComputed          = "rl_reward_computed"
	// EventTypePolicyRejected is emitted when a stored policy fails the
	// architecture check and the agent is not run.
	EventTypePolicyRejected = "rl_policy_rejected"

	AttributeKeyHeight    = "height"
	AttributeKeyEpoch     = "epoch"
	AttributeKeyAgentMode = "agent_mode"
	AttributeKeyReward    = "reward"
	AttributeKeyReason    = "reason"
)
