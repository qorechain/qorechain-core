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
	// FeatureEVMPQCAuthorisationWindow makes the post-quantum requirement true
	// on the EVM lane: an EVM transaction is admitted only from an account that
	// holds a registered post-quantum key and has opened a bounded
	// authorisation window with a Cosmos-lane message. The window messages,
	// queries and CLI ship with the binary and work regardless; only the
	// refusal in the ante path waits for this switch.
	FeatureEVMPQCAuthorisationWindow = "evm-pqc-authorisation-window"
)

// AllProtocolFeatures lists every feature this binary ENABLES: a fresh chain
// enables all of them at genesis, and the release upgrade handler enables
// them at the plan height. A feature that is not yet released is listed in
// StagedProtocolFeatures instead: the code is in the binary, the switch stays
// off on every chain, and a later release's handler turns it on.
var AllProtocolFeatures = []string{
	FeatureEVMLockedBalanceScaling,
	FeatureRedelegationSlashCap,
	FeatureInflationEpochScaling,
	FeatureBurnStakerShareToStakers,
	FeatureAITxRateLimit,
}

// StagedProtocolFeatures are known to this binary and enabled by no path in
// it. Moving one to AllProtocolFeatures is what releases it.
var StagedProtocolFeatures = []string{
	FeatureEVMPQCAuthorisationWindow,
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
