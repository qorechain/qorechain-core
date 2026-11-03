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
	// verification, a price set from a Mac measurement where ML-DSA-87 is
	// about 1.6 times a secp256k1 verification; on the fleet's linux/amd64
	// hosts it is about 7 times (0.29 ms against about 0.04 ms, profiled on
	// diana), so a verification was paid at under a third of what the SDK
	// asks for the same CPU time on the classical side. The key registration,
	// rotation and migration proofs and every bridge verification charged
	// nothing at all.
	FeaturePQCVerifyGasCalibrated = "pqc-verify-gas-calibrated"
)

// PQCVerifyGasCalibrated is the price of one ML-DSA-87 verification from
// FeaturePQCVerifyGasCalibrated. It keeps the SDK's own anchor, 1000 gas for
// a secp256k1 verification, at the ratio measured where the chain runs:
// through the library the ante calls, ML-DSA-87 verification takes about
// 0.29 ms on a fleet host against about 0.04 ms for secp256k1, so about
// 6900 gas, rounded to 7000. The signature's bytes are paid for separately
// by the transaction size charge (4627 bytes, 46270 gas at 10 a byte).
const PQCVerifyGasCalibrated uint64 = 7000

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
