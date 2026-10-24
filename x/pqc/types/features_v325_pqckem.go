package types

// v3.2.5 post-quantum KEM registry and key-maintenance switches. Below the
// v3.2.5 plan height every one of them is off, so history replays
// byte-identically.
const (
	// FeaturePQCKEMRegistry opens the native KEM key registry: an account
	// that holds a post-quantum signature key may publish one ML-KEM (FIPS
	// 203) encapsulation key with MsgRegisterKEMKey, replace it by sending
	// the message again, and withdraw it with MsgRemoveKEMKey. Both messages
	// need the account's hybrid signature and are refused inside authz. The
	// messages, the query and the CLI ship with the binary; the handler
	// refuses them until this switch is on.
	FeaturePQCKEMRegistry = "pqc-kem-registry"
	// FeaturePQCRotateKeySizeCheck applies the stateless size rule of a
	// rotation (the new key has the old key's length, QSR-2026-0607) in block
	// execution, before the fee is charged, the way ValidateBasic would. The
	// handler always refused such a rotation, but only after the fee.
	// ValidateBasic itself cannot carry the rule: it has no height, and it
	// runs in block execution, so tightening it would change which historical
	// transactions were valid.
	FeaturePQCRotateKeySizeCheck = "pqc-rotate-key-size-check"
)

// V325PQCKEMFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325PQCKEMFeatures = []string{
	FeaturePQCKEMRegistry,
	FeaturePQCRotateKeySizeCheck,
}
