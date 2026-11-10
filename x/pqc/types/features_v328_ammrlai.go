package types

// v3.2.8 switches for the AMM, gas abstraction, x/rlconsensus and x/ai end to
// end item. Each one changes state, gas or the outcome of a message only from
// the height the v3.2.8 release handler enables it; below it every path runs
// the historical code.
const (
	// FeatureAMMRespectsSendEnabled makes the AMM honour the bank module's
	// send-enabled setting. Pool creation, adding and removing liquidity and
	// both swap directions are refused when a coin they would move, on either
	// leg, has sending disabled. Before it the AMM moved coins through the
	// module-account helpers, which do not consult that setting, so a denom
	// whose transfers were switched off could still change hands through a
	// pool.
	FeatureAMMRespectsSendEnabled = "amm-respects-send-enabled"

	// FeatureGasAbstractionDepthAgeGate tightens the conditions under which an
	// x/amm pool may price fees paid in a foreign token (price source
	// amm_twap). The pool must also hold min_foreign_liquidity of the foreign
	// token, the listing must name that minimum, and the pool must be at least
	// max(min_pool_age_blocks, MinFeePricingPoolAgeBlocks) blocks old, so its
	// running average has a history of its own. The float refill prices with
	// the same rules as the fee path (constant_product pools only). Before it
	// only the native side of the pool was checked, a pool could price fees
	// from the block after it was created, and the refill accepted any pool
	// type. No foreign fee token is listed on either network.
	FeatureGasAbstractionDepthAgeGate = "gasabstraction-depth-age-gate"

	// FeatureAMMGovParams lets governance change the AMM parameters with
	// MsgUpdateParams (authority: the gov module account). Before it the
	// parameters could only change in an upgrade handler.
	FeatureAMMGovParams = "amm-gov-params"

	// FeatureRLObservationInputs fills the agent's observation from chain
	// state that was wired but never read: validator count, reputation mean,
	// spread and Gini (x/reputation), x/ai anomalies detected during the step,
	// the EVM base fee, the share of validators absent from the last commit,
	// the bonded ratio and the gas used over the step against the block gas
	// limit (gas is summed in EndBlock, where it is known). The rlConsensusParams
	// precompile reports the bonded validator count. Before it those
	// dimensions stayed zero or held placeholders, and block utilization was
	// read in BeginBlock, before any transaction had used gas.
	FeatureRLObservationInputs = "rl-observation-inputs"

	// FeatureRLRelativeBounds makes max_change a relative bound for the block
	// time target and the gas price floor (at most max_change times the
	// current value per step, so 10% in conservative mode) instead of an
	// absolute one, which truncated every block time change to zero
	// milliseconds and moved the gas price floor by at most 0.1. The circuit
	// breaker's recovery is counted in blocks, as its window is. Before it the
	// recovery counted agent steps against a block count, ten times too long.
	// The outputs stay inside the module's own store and the existing hard
	// limits (1 to 30 seconds, gas price floor at least 1, pool weights 0.05
	// to 0.80).
	FeatureRLRelativeBounds = "rl-relative-bounds"

	// FeatureEVMAIRiskScoreBytecode makes the aiRiskScore precompile (0x0B01)
	// score the EVM bytecode it is given: it walks the opcodes (skipping PUSH
	// data) and scores self-destruct, delegatecall, callcode and create2, an
	// EIP-7702 delegation designator and code above the EIP-170 size limit, in
	// integer basis points. Before it only the length of the argument was
	// looked at. 0x0B01 is not an active precompile on either network.
	FeatureEVMAIRiskScoreBytecode = "evm-ai-risk-score-bytecode"

	// FeatureAIFlaggedTxRetention bounds the x/ai flagged-transaction records:
	// each record is also indexed by height, and every new record deletes up
	// to AIFlaggedPrunePerWrite records older than AIFlaggedRetentionBlocks,
	// and up to the same number of records written before the switch (which
	// have no height index). Before it a record was kept forever.
	FeatureAIFlaggedTxRetention = "ai-flagged-tx-retention"
)

// V328AMMRLAIFeatures are the v3.2.8 switches of the AMM, RL consensus and AI
// end to end item.
var V328AMMRLAIFeatures = []string{
	FeatureAMMRespectsSendEnabled,
	FeatureGasAbstractionDepthAgeGate,
	FeatureAMMGovParams,
	FeatureRLObservationInputs,
	FeatureRLRelativeBounds,
	FeatureEVMAIRiskScoreBytecode,
	FeatureAIFlaggedTxRetention,
}
