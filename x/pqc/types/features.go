package types

// Protocol features are behaviour switches that must turn on at one height on
// every node, and stay on. They exist for fixes that live in upstream
// dependencies (patched at build time) and cannot consult our upgrade handler
// themselves: the patched code asks the application, the application asks
// this record.
//
// A chain that upgrades into a release enables the features in that release's
// upgrade handler at the plan height; a chain that starts on the release
// enables them in InitGenesis. Below the plan height nothing is enabled, so
// history replays byte-identically under the new binary.
const (
	// FeatureEVMLockedBalanceScaling keeps the EVM state object's locked
	// balance snapshot in the EVM's decimal representation, the same one its
	// spendable balance uses.
	FeatureEVMLockedBalanceScaling = "evm-locked-balance-scaling"

	// FeatureRedelegationSlashCap bounds a redelegation entry's total slash at
	// slashFactor * InitialBalance across the destination's unbonding entries
	// and live delegation.
	FeatureRedelegationSlashCap = "staking-redelegation-slash-cap"
	// FeatureInflationEpochScaling spreads schedule-mode emission over the
	// year by block count, so an epoch mints the share of the year its length
	// represents instead of a full day regardless of length.
	FeatureInflationEpochScaling = "inflation-epoch-scaling"
	// FeatureBurnStakerShareToStakers leaves the staker share of fees in the
	// fee collector, where the distribution module pays it out, instead of
	// depositing it in the distribution module's balance where nobody could
	// claim it.
	FeatureBurnStakerShareToStakers = "burn-staker-share-to-stakers"
	// FeatureAITxRateLimit enforces the declared per-sender transaction rate
	// limit in the ante path.
	FeatureAITxRateLimit = "ai-tx-rate-limit"
	// FeaturePQCSignBytesV2 switches every post-quantum signed payload the
	// chain verifies (hybrid transaction sign-bytes, key-migration preimage,
	// bridge attestation) from its first form to the domain-tagged,
	// chain-bound v2 form. Below the switch the chain verifies exactly what
	// wallets signed historically; from it, only v2. No block accepts both.
	FeaturePQCSignBytesV2 = "pqc-sign-bytes-v2"
	// FeatureEVMPQCAuthorisationWindow makes the post-quantum requirement true
	// on the EVM lane: an EVM transaction is admitted only from an account that
	// holds a registered post-quantum key and has opened a bounded
	// authorisation window with a Cosmos-lane message. The window messages,
	// queries and CLI ship with the binary and work regardless; only the
	// refusal in the ante path waits for this switch.
	FeatureEVMPQCAuthorisationWindow = "evm-pqc-authorisation-window"
)

// Released in v3.2.2. Each switch guards one fix from the bug-bounty
// re-verification of 2026-09-29; below the v3.2.2 plan height every one of
// them is off, so history replays byte-identically.
const (
	// EIP-7702: an authority's delegation needs its own post-quantum consent.
	FeatureEVM7702AuthorityConsent = "evm-7702-authority-consent"
	// EIP-7702: value leaving a delegated account is charged to its own window.
	FeatureEVMDelegatedActorSettlement = "evm-delegated-actor-settlement"
	// EIP-7702: a failed type-4 transaction keeps the authorizations it applied.
	FeatureEVM7702FailedTxAuthorizations = "evm-7702-failed-tx-authorizations"
	// Multisig signatures are bounded in block execution too.
	FeatureMultisigSignatureBound = "multisig-signature-bound"
	// Validator licence and self-bond floor are enforced by the staking message
	// server, on every dispatch path.
	FeatureStakingAdmissionGate = "staking-admission-gate"
	// AMM swaps bind both denominations to the pool and price stable pools with
	// the stable invariant in both directions.
	FeatureAMMSwapIntegrity = "amm-swap-integrity"
	// AMM protocol fee is collected and routed.
	FeatureAMMProtocolFee = "amm-protocol-fee"
	// AMM max_swap_impact_bps is enforced.
	FeatureAMMSwapImpactCap = "amm-swap-impact-cap"
	// Abstract-account rule-bound keys: deny-by-default EVM calls, measured outflow,
	// per-token caps, deploy permission for CREATE.
	FeatureAASpendingPolicy = "abstractaccount-spending-policy"
	// EVM: unprotected (pre-EIP-155) transactions refused in block execution.
	FeatureEVMUnprotectedTxRefusal = "evm-unprotected-tx-refusal"
	// EVM: contract creation keeps the sender nonce increments made during it.
	FeatureEVMCreateNoncePreserve = "evm-create-nonce-preserve"
	// ERC-20 IBC middleware: reconversion on timeout/error-ack is isolated.
	FeatureERC20IBCRefundIsolation = "erc20-ibc-refund-isolation"
	// Module account addresses cannot be squatted through the EVM.
	FeatureModuleAccountGuard = "module-account-guard"
	// Bridge: zero amounts and configuration are validated consistently.
	FeatureBridgeValidation = "bridge-validation"
	// Burn: the failed_tx burn channel applies.
	FeatureBurnFailedTx = "burn-failed-tx"
	// Inflation: an emission cap of zero or omitted is refused, not unlimited.
	FeatureInflationEmissionCapGuard = "inflation-emission-cap-guard"
	// Governance proposer reward is bounded.
	FeatureGovProposerRewardCap = "gov-proposer-reward-cap"
	// PQC key registration validates ecdsa_pubkey and binds it to the signer.
	FeaturePQCECDSAPubkeyValidation = "pqc-ecdsa-pubkey-validation"
	// x/multilayer MsgUpdateParams is routable.
	FeatureMultilayerParamsRoute = "multilayer-params-route"
	// RDK: rollup and layer ids are validated; finalization keeps the batch pointer.
	FeatureRDKIntegrity = "rdk-integrity"
	// SVM: base58 addresses are bounded and canonical.
	FeatureSVMBase58Canonical = "svm-base58-canonical"
	// EVM lane runs the QoreChain ante checks it used to skip (burn counter,
	// AI anomaly and rate limit).
	FeatureEVMAnteParity = "evm-ante-parity"
	// Gas abstraction: documented non-native fee denoms are accepted.
	FeatureGasAbstractionFees = "gas-abstraction-fees"
	// Reputation: incremental EndBlocker, live stake and contribution scores.
	FeatureReputationScoring = "reputation-scoring"
	// rlconsensus: policy validation with checked arithmetic.
	FeatureRLPolicyValidation = "rlconsensus-policy-validation"
	// x/ai anomaly detection and rate limit apply to standard Cosmos messages.
	FeatureAICosmosAnomaly = "ai-cosmos-anomaly"
)

// ReleaseV3_2_0ProtocolFeatures is exactly what the v3.1.98 / v3.2.0 release
// handler enabled when it ran. It is frozen: that handler must enable the same
// set when history is replayed, whatever later releases add.
var ReleaseV3_2_0ProtocolFeatures = []string{
	FeatureEVMLockedBalanceScaling,
	FeatureRedelegationSlashCap,
	FeatureInflationEpochScaling,
	FeatureBurnStakerShareToStakers,
	FeatureAITxRateLimit,
	FeaturePQCSignBytesV2,
	FeatureEVMPQCAuthorisationWindow,
}

// ReleaseV3_2_2ProtocolFeatures is what the v3.2.2 release handler enables at
// its plan height.
var ReleaseV3_2_2ProtocolFeatures = []string{
	FeatureEVM7702AuthorityConsent,
	FeatureEVMDelegatedActorSettlement,
	FeatureEVM7702FailedTxAuthorizations,
	FeatureMultisigSignatureBound,
	FeatureStakingAdmissionGate,
	FeatureAMMSwapIntegrity,
	FeatureAMMProtocolFee,
	FeatureAMMSwapImpactCap,
	FeatureAASpendingPolicy,
	FeatureEVMUnprotectedTxRefusal,
	FeatureEVMCreateNoncePreserve,
	FeatureERC20IBCRefundIsolation,
	FeatureModuleAccountGuard,
	FeatureBridgeValidation,
	FeatureBurnFailedTx,
	FeatureInflationEmissionCapGuard,
	FeatureGovProposerRewardCap,
	FeaturePQCECDSAPubkeyValidation,
	FeatureMultilayerParamsRoute,
	FeatureRDKIntegrity,
	FeatureSVMBase58Canonical,
	FeatureEVMAnteParity,
	FeatureGasAbstractionFees,
	FeatureReputationScoring,
	FeatureRLPolicyValidation,
	FeatureAICosmosAnomaly,
}

// Released in v3.2.3.
const (
	// x/license: governance can replace the licence grant authority.
	FeatureLicenseAuthorityUpdate = "license-authority-update"
)

// ReleaseV3_2_3ProtocolFeatures is what the v3.2.3 release handler enables at
// its plan height.
var ReleaseV3_2_3ProtocolFeatures = concatFeatures(
	[]string{FeatureLicenseAuthorityUpdate},
	V323PqcIntegrityFeatures,
	V323EvmWindowFeatures,
	V323RdkFeatures,
	V323CrossvmPrecompileFeatures,
	V323StakingVestingFeatures,
	V323EconFeatures,
	V323HardeningFeatures,
)

// AllProtocolFeatures lists every feature this binary ENABLES: a fresh chain
// enables all of them at genesis, and the release upgrade handler enables
// them at the plan height. A feature that is not yet released is listed in
// StagedProtocolFeatures instead: the code is in the binary, the switch stays
// off on every chain, and a later release's handler turns it on.
var AllProtocolFeatures = concatFeatures(ReleaseV3_2_0ProtocolFeatures, ReleaseV3_2_2ProtocolFeatures, ReleaseV3_2_3ProtocolFeatures)

// StagedProtocolFeatures are known to this binary and enabled by no path in
// it. Moving one to AllProtocolFeatures is what releases it.
var StagedProtocolFeatures = []string{}

func concatFeatures(lists ...[]string) []string {
	var out []string
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

// KnownProtocolFeature reports whether this binary knows the feature at all,
// released or staged.
func KnownProtocolFeature(name string) bool {
	for _, f := range AllProtocolFeatures {
		if f == name {
			return true
		}
	}
	for _, f := range StagedProtocolFeatures {
		if f == name {
			return true
		}
	}
	return false
}

// ProtocolFeaturePrefix is the store prefix under which enabled features are
// recorded, one key per feature, value 0x01.
var ProtocolFeaturePrefix = []byte("pqc/features/")

// ProtocolFeatureKey returns the store key for a feature.
func ProtocolFeatureKey(name string) []byte {
	return append(append([]byte{}, ProtocolFeaturePrefix...), name...)
}
