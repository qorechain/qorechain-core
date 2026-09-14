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
)

// AllProtocolFeatures lists every feature this binary knows. A fresh chain
// enables all of them at genesis; the release upgrade handler enables the
// ones it introduces.
var AllProtocolFeatures = []string{
	FeatureEVMLockedBalanceScaling,
	FeatureRedelegationSlashCap,
}

// ProtocolFeaturePrefix is the store prefix under which enabled features are
// recorded, one key per feature, value 0x01.
var ProtocolFeaturePrefix = []byte("pqc/features/")

// ProtocolFeatureKey returns the store key for a feature.
func ProtocolFeatureKey(name string) []byte {
	return append(append([]byte{}, ProtocolFeaturePrefix...), name...)
}
