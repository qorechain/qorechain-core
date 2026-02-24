package precompiles

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Function selectors (first 4 bytes of keccak256 of Solidity signature).
var (
	// pqcVerify(bytes,bytes,bytes)
	SelectorPQCVerify [4]byte
	// pqcKeyStatus(address)
	SelectorPQCKeyStatus [4]byte
	// aiRiskScore(bytes)
	SelectorAIRiskScore [4]byte
	// aiAnomalyCheck(address,uint256)
	SelectorAIAnomalyCheck [4]byte
	// rlConsensusParams()
	SelectorRLConsensusParams [4]byte
	// executeCrossVMCall(uint8,string,bytes)
	SelectorCrossVMCall [4]byte
)

func init() {
	SelectorPQCVerify = computeSelector("pqcVerify(bytes,bytes,bytes)")
	SelectorPQCKeyStatus = computeSelector("pqcKeyStatus(address)")
	SelectorAIRiskScore = computeSelector("aiRiskScore(bytes)")
	SelectorAIAnomalyCheck = computeSelector("aiAnomalyCheck(address,uint256)")
	SelectorRLConsensusParams = computeSelector("rlConsensusParams()")
	SelectorCrossVMCall = computeSelector("executeCrossVMCall(uint8,string,bytes)")
}

func computeSelector(signature string) [4]byte {
	hash := crypto.Keccak256([]byte(signature))
	var sel [4]byte
	copy(sel[:], hash[:4])
	return sel
}

// DecodePQCVerifyInput decodes pqcVerify(bytes pubkey, bytes sig, bytes message).
func DecodePQCVerifyInput(input []byte) ([]byte, []byte, []byte, error) {
	if len(input) < 4 {
		return nil, nil, nil, fmt.Errorf("input too short: %d bytes", len(input))
	}
	args := input[4:]

	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: bytesType},
		{Type: bytesType},
		{Type: bytesType},
	}
	values, err := arguments.Unpack(args)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ABI decode failed: %w", err)
	}
	return values[0].([]byte), values[1].([]byte), values[2].([]byte), nil
}

// EncodePQCVerifyOutput encodes the output for pqcVerify -> (bool).
func EncodePQCVerifyOutput(valid bool) ([]byte, error) {
	boolType, _ := abi.NewType("bool", "", nil)
	arguments := abi.Arguments{{Type: boolType}}
	return arguments.Pack(valid)
}

// DecodePQCKeyStatusInput decodes pqcKeyStatus(address).
func DecodePQCKeyStatusInput(input []byte) (common.Address, error) {
	if len(input) < 4 {
		return common.Address{}, fmt.Errorf("input too short")
	}
	addressType, _ := abi.NewType("address", "", nil)
	arguments := abi.Arguments{{Type: addressType}}
	values, err := arguments.Unpack(input[4:])
	if err != nil {
		return common.Address{}, fmt.Errorf("ABI decode failed: %w", err)
	}
	return values[0].(common.Address), nil
}

// EncodePQCKeyStatusOutput encodes (bool registered, uint8 algorithmId, bytes pubkey).
func EncodePQCKeyStatusOutput(registered bool, algorithmId uint8, pubkey []byte) ([]byte, error) {
	boolType, _ := abi.NewType("bool", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: boolType},
		{Type: uint8Type},
		{Type: bytesType},
	}
	return arguments.Pack(registered, algorithmId, pubkey)
}

// DecodeAIRiskScoreInput decodes aiRiskScore(bytes txData).
func DecodeAIRiskScoreInput(input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, fmt.Errorf("input too short")
	}
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{{Type: bytesType}}
	values, err := arguments.Unpack(input[4:])
	if err != nil {
		return nil, fmt.Errorf("ABI decode failed: %w", err)
	}
	return values[0].([]byte), nil
}

// EncodeAIRiskScoreOutput encodes (uint256 score, uint8 level).
func EncodeAIRiskScoreOutput(score *big.Int, level uint8) ([]byte, error) {
	uint256Type, _ := abi.NewType("uint256", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	arguments := abi.Arguments{
		{Type: uint256Type},
		{Type: uint8Type},
	}
	return arguments.Pack(score, level)
}

// DecodeAIAnomalyCheckInput decodes aiAnomalyCheck(address, uint256).
func DecodeAIAnomalyCheckInput(input []byte) (common.Address, *big.Int, error) {
	if len(input) < 4 {
		return common.Address{}, nil, fmt.Errorf("input too short")
	}
	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	arguments := abi.Arguments{
		{Type: addressType},
		{Type: uint256Type},
	}
	values, err := arguments.Unpack(input[4:])
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("ABI decode failed: %w", err)
	}
	return values[0].(common.Address), values[1].(*big.Int), nil
}

// EncodeAIAnomalyCheckOutput encodes (uint256 anomalyScore, bool flagged).
func EncodeAIAnomalyCheckOutput(anomalyScore *big.Int, flagged bool) ([]byte, error) {
	uint256Type, _ := abi.NewType("uint256", "", nil)
	boolType, _ := abi.NewType("bool", "", nil)
	arguments := abi.Arguments{
		{Type: uint256Type},
		{Type: boolType},
	}
	return arguments.Pack(anomalyScore, flagged)
}

// DecodeCrossVMCallInput decodes executeCrossVMCall(uint8 targetVM, string targetContract, bytes payload).
func DecodeCrossVMCallInput(input []byte) (uint8, string, []byte, error) {
	if len(input) < 4 {
		return 0, "", nil, fmt.Errorf("input too short")
	}
	uint8Type, _ := abi.NewType("uint8", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: uint8Type},
		{Type: stringType},
		{Type: bytesType},
	}
	values, err := arguments.Unpack(input[4:])
	if err != nil {
		return 0, "", nil, fmt.Errorf("ABI decode failed: %w", err)
	}
	return values[0].(uint8), values[1].(string), values[2].([]byte), nil
}

// EncodeRLConsensusParamsOutput encodes (uint256 blockTime, uint256 baseGasPrice, uint256 validatorSetSize, uint256 epoch).
func EncodeRLConsensusParamsOutput(blockTime, baseGasPrice, validatorSetSize, epoch *big.Int) ([]byte, error) {
	uint256Type, _ := abi.NewType("uint256", "", nil)
	arguments := abi.Arguments{
		{Type: uint256Type},
		{Type: uint256Type},
		{Type: uint256Type},
		{Type: uint256Type},
	}
	return arguments.Pack(blockTime, baseGasPrice, validatorSetSize, epoch)
}
