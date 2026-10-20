package types

// Released in v3.2.3.
const (
	// FeatureEVMPQCWindowTargets honours the optional scope of an EVM-lane
	// authorisation window: native_only (plain native QOR transfers only) and
	// allowed_targets (listed recipients only). Below the switch a window
	// cannot be opened with a scope, and a window without one admits exactly
	// what it always did (QSR-2026-0347, -0552).
	FeatureEVMPQCWindowTargets = "evm-pqc-window-targets"
)

// V323EvmWindowFeatures are the v3.2.3 features of the EVM window scope fix.
var V323EvmWindowFeatures = []string{FeatureEVMPQCWindowTargets}
