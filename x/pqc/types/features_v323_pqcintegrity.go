package types

// v3.2.3 post-quantum key integrity switches (QSR-2026-0343, 0550, 0400, 0362,
// 0538, 0506, 0521, 0372, 0487, 0453, 0514). Below the v3.2.3 plan height
// every one of them is off, so history replays byte-identically.
const (
	// An account that already holds a post-quantum key may send key
	// maintenance without the hybrid signature only to repair a record that
	// holds no usable key. Before this switch a classical-only key could make
	// a register or migrate from a keyed account pass the ante, so the fee was
	// taken and the handler then failed.
	FeaturePQCKeyedAccountMaintenanceHybrid = "pqc-keyed-account-maintenance-hybrid"
	// Registration refuses classical_only, empty and degenerate (t1 all zero)
	// keys, verifies an optional proof of possession, and lets a record that
	// holds no usable key be replaced by MsgRegisterPQCKeyV2.
	FeaturePQCRegistrationIntegrity = "pqc-registration-integrity"
	// Rotation and migration refuse a key the account used before, and the
	// key-management messages cannot be sent through authz.
	FeaturePQCRotationReplayGuard = "pqc-rotation-replay-guard"
	// Deprecating or disabling an algorithm cannot leave the chain without an
	// active signature algorithm, and the replacement must be able to sign.
	FeaturePQCAlgorithmGovernanceGuards = "pqc-algorithm-governance-guards"
	// A transaction carries at most one hybrid signature extension.
	FeaturePQCHybridSingleExtension = "pqc-hybrid-single-extension"
	// Under HybridRequired, a keyless account cannot pass the ante by
	// attaching a hybrid extension that has no registered key to verify
	// against; only a registration may pass, as with no extension at all.
	FeaturePQCKeylessExtensionClosed = "pqc-keyless-extension-closed"
)

// V323PqcIntegrityFeatures is this area's share of the v3.2.3 release set.
var V323PqcIntegrityFeatures = []string{
	FeaturePQCKeyedAccountMaintenanceHybrid,
	FeaturePQCRegistrationIntegrity,
	FeaturePQCRotationReplayGuard,
	FeaturePQCAlgorithmGovernanceGuards,
	FeaturePQCHybridSingleExtension,
	FeaturePQCKeylessExtensionClosed,
}
