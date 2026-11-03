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
	// FeaturePQCVerifyOncePerSigner makes the two PQC ante decorators judge
	// each signer of a transaction once. Before it they ran their whole
	// signer loop once per message, so a transaction of M messages paid for
	// M post-quantum verifications and M sets of counter writes for one
	// signature, and a getSigners error was ignored. From the switch the
	// signers are read once, an error refuses the transaction, and every
	// hybrid verification is charged PQCHybridVerifyGas before it runs, in
	// CheckTx, simulation and block execution alike (audit 2026-10-25,
	// PQC-01).
	FeaturePQCVerifyOncePerSigner = "pqc-verify-once-per-signer"
)

// PQCHybridVerifyGas is what one ML-DSA-87 hybrid verification costs from
// FeaturePQCVerifyOncePerSigner. Calibrated against the SDK's own anchor,
// 1000 gas for a secp256k1 verification: measured on one core, ML-DSA-87
// verification through the library takes about 125 microseconds against
// about 84 for secp256k1 (and 33 for ed25519, which the SDK prices at 590),
// so the CPU-equivalent price is about 1500. It is set at 2000 to leave room
// for slower hosts. The signature's bytes are already paid for by the
// transaction size charge. From FeaturePQCVerifyGasCalibrated (v3.2.7) the
// price is PQCVerifyGasCalibrated: on the fleet's linux/amd64 hosts the
// ratio to secp256k1 is about 7, not 1.6.
const PQCHybridVerifyGas uint64 = 2000

// V325PQCKEMFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325PQCKEMFeatures = []string{
	FeaturePQCKEMRegistry,
	FeaturePQCRotateKeySizeCheck,
	FeaturePQCVerifyOncePerSigner,
}
