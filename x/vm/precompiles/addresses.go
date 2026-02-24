package precompiles

import "github.com/ethereum/go-ethereum/common"

// Precompile addresses for QoreChain custom precompiles.
// These are statically allocated in the EVM address space.
var (
	// CrossVMBridgeAddress is the precompile for cross-VM calls (EVM <-> CosmWasm).
	CrossVMBridgeAddress = common.HexToAddress("0x0000000000000000000000000000000000000901")

	// PQCVerifyAddress is the precompile for post-quantum signature verification.
	PQCVerifyAddress = common.HexToAddress("0x0000000000000000000000000000000000000A01")

	// PQCKeyStatusAddress is the precompile for PQC key registration queries.
	PQCKeyStatusAddress = common.HexToAddress("0x0000000000000000000000000000000000000A02")

	// AIRiskScoreAddress is the precompile for AI risk assessment.
	AIRiskScoreAddress = common.HexToAddress("0x0000000000000000000000000000000000000B01")

	// AIAnomalyCheckAddress is the precompile for anomaly detection.
	AIAnomalyCheckAddress = common.HexToAddress("0x0000000000000000000000000000000000000B02")

	// RLConsensusParamsAddress is the precompile for RL-tuned consensus parameters.
	RLConsensusParamsAddress = common.HexToAddress("0x0000000000000000000000000000000000000C01")
)
