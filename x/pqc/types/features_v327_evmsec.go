package types

const (
	// FeatureEVMPrecompileCallLimitExact makes the per-transaction limit of 20
	// stateful precompile calls exact (cosmos-evm patch 0062,
	// GHSA-wm4f-qpxp-rch4): a refused call no longer advances the counter, so
	// it cannot wrap at 256 and reopen the limit, and the refused call's
	// multistore snapshot is popped. Below it the counter wraps as before.
	FeatureEVMPrecompileCallLimitExact = "evm-precompile-call-limit-exact"

	// FeatureEVMInternalGasAccounting caps a module's internal EVM call at the
	// gas the transaction still has, and charges the gas an execution used
	// when its state write fails afterwards (cosmos-evm patch 0063,
	// GHSA-p8x8-4gg6-prcc). Below it the internal call runs with 25M gas and
	// a failed write charges nothing for the execution.
	FeatureEVMInternalGasAccounting = "evm-internal-gas-accounting"

	// FeatureEVMSelfDestructStakingGuard refuses the self-destruct of a
	// contract that holds delegations, redelegations or unbonding
	// delegations (cosmos-evm patch 0064, GHSA-c6xv-pppp-rjw6). The check
	// reads the staking store, so every self-destruct commit costs more gas
	// from the switch.
	FeatureEVMSelfDestructStakingGuard = "evm-selfdestruct-staking-guard"
)

// V327EVMSecFeatures are the v3.2.7 switches for the November 2026 cosmos/evm
// advisories.
var V327EVMSecFeatures = []string{
	FeatureEVMPrecompileCallLimitExact,
	FeatureEVMInternalGasAccounting,
	FeatureEVMSelfDestructStakingGuard,
}
