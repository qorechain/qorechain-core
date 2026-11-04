package types

// v3.2.7 switch that prices post-quantum signature verification by its real
// cost. Listed in V327PQCGasFeatures below.
const (
	// FeaturePQCVerifyGasCalibrated charges PQCVerifyGasCalibrated for every
	// ML-DSA verification the chain runs inside a transaction, before it
	// runs, the same in CheckTx, simulation and block execution: the hybrid
	// signature of each keyed signer (PQCHybridVerifyDecorator), each member
	// signature of a post-quantum multisig, each member proof of possession
	// of MsgRegisterPQCMultisig, the proof of possession of
	// MsgRegisterPQCKeyV2, the two signatures of MsgRotatePQCKey and of
	// MsgMigratePQCKey, and in x/bridge the attestation signature, the
	// re-verification of each stored attestation, each deposit attestation
	// and the bridge validator's proof of possession.
	//
	// Before it the ante paths charged PQCHybridVerifyGas (2000) per
	// verification, but the key registration, rotation and migration proofs
	// and every bridge verification charged nothing at all. Measured on a
	// fleet host (Xeon 8375C, linux/amd64, through the library the ante
	// calls), ML-DSA-87 verification takes about 297 microseconds against
	// about 166 for secp256k1, a ratio of about 1.8, so 2000 against the
	// SDK's 1000 for secp256k1 already covers it; the switch keeps that price
	// and charges it everywhere a verification runs.
	FeaturePQCVerifyGasCalibrated = "pqc-verify-gas-calibrated"
)

// PQCVerifyGasCalibrated is the price of one ML-DSA-87 verification from
// FeaturePQCVerifyGasCalibrated. It keeps the SDK's own anchor, 1000 gas for
// a secp256k1 verification, at the ratio measured on a fleet host (about 297
// against 166 microseconds, 1.8), rounded up to 2000, the price the ante
// already charged. The signature's bytes are paid for separately by the
// transaction size charge (4627 bytes, 46270 gas at 10 a byte).
const PQCVerifyGasCalibrated uint64 = 2000

// PQCVerifyGas is the price of one post-quantum verification: from the
// calibration switch PQCVerifyGasCalibrated, before it PQCHybridVerifyGas,
// what the ante paths charged. A site that charged nothing before the switch
// checks the switch itself and charges only from it.
func PQCVerifyGas(calibrated bool) uint64 {
	if calibrated {
		return PQCVerifyGasCalibrated
	}
	return PQCHybridVerifyGas
}

// V327PQCGasFeatures are the v3.2.7 switches that price post-quantum signature verification by its real cost.
var V327PQCGasFeatures = []string{
	FeaturePQCVerifyGasCalibrated,
}
