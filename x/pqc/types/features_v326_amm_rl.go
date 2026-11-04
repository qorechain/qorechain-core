package types

// v3.2.6 x/rlconsensus switches. They are listed in V326Features
// (features_v326.go), one line each.
const (
	// FeatureRLAgentObservable records the action the agent proposes at each
	// step, in every mode including shadow, under its own store key, and emits
	// it as an rl_action_proposed event. Before it a shadow step left nothing
	// on chain but the step counter: the proposed values only reached the
	// node's log. It writes one key per step and changes no other state.
	FeatureRLAgentObservable = "rl-agent-observable"
	// FeatureRLCircuitBreakerPerBlock makes the agent's circuit breaker judge
	// the average block time over the observation interval. Before it the
	// breaker compared the gap between two agent steps, which spans
	// observation_interval blocks, against twice the per-block target, so with
	// the default interval of 10 it tripped on every second step of a healthy
	// chain and the agent ran about once every 1,000 blocks.
	FeatureRLCircuitBreakerPerBlock = "rl-circuit-breaker-per-block"
	// FeatureRLHistoryRetention bounds the observation and reward history:
	// records older than RLHistoryRetentionSteps steps are deleted, at most
	// RLHistoryPrunePerStep of each kind per step, so the backlog kept since
	// genesis drains over a few days without a heavy block. Before it one
	// observation and one reward were added every observation_interval blocks
	// and never removed.
	FeatureRLHistoryRetention = "rl-history-retention"
)
