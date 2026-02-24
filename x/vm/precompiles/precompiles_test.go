package precompiles

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestStubPrecompilesAddress verifies each stub precompile returns its correct address.
func TestStubPrecompilesAddress(t *testing.T) {
	tests := []struct {
		name     string
		precomp  interface{ Address() common.Address }
		expected common.Address
	}{
		{"PQCVerify", &PQCVerifyPrecompile{}, PQCVerifyAddress},
		{"PQCKeyStatus", &PQCKeyStatusPrecompile{}, PQCKeyStatusAddress},
		{"AIRiskScore", &AIRiskScorePrecompile{}, AIRiskScoreAddress},
		{"AIAnomalyCheck", &AIAnomalyCheckPrecompile{}, AIAnomalyCheckAddress},
		{"RLConsensusParams", &RLConsensusParamsPrecompile{}, RLConsensusParamsAddress},
		{"CrossVMBridge", &CrossVMBridgePrecompile{}, CrossVMBridgeAddress},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.precomp.Address()
			if got != tc.expected {
				t.Errorf("Address() = %s, want %s", got.Hex(), tc.expected.Hex())
			}
		})
	}
}

// TestStubPrecompilesGas verifies each stub precompile returns correct gas costs.
func TestStubPrecompilesGas(t *testing.T) {
	tests := []struct {
		name        string
		precomp     interface{ RequiredGas([]byte) uint64 }
		input       []byte
		expectedGas uint64
	}{
		{"PQCVerify_empty", &PQCVerifyPrecompile{}, nil, 25_000},
		{"PQCVerify_100bytes", &PQCVerifyPrecompile{}, make([]byte, 100), 25_000 + 100*8},
		{"PQCKeyStatus", &PQCKeyStatusPrecompile{}, nil, 2_500},
		{"AIRiskScore", &AIRiskScorePrecompile{}, nil, 50_000},
		{"AIAnomalyCheck", &AIAnomalyCheckPrecompile{}, nil, 40_000},
		{"RLConsensusParams", &RLConsensusParamsPrecompile{}, nil, 1_500},
		{"CrossVMBridge", &CrossVMBridgePrecompile{}, nil, 50_000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.precomp.RequiredGas(tc.input)
			if got != tc.expectedGas {
				t.Errorf("RequiredGas() = %d, want %d", got, tc.expectedGas)
			}
		})
	}
}

// TestPQCVerifyGasScaling verifies PQC verify gas scales with input size.
func TestPQCVerifyGasScaling(t *testing.T) {
	p := &PQCVerifyPrecompile{}

	sizes := []int{0, 10, 100, 1000, 10000}
	for _, size := range sizes {
		input := make([]byte, size)
		expected := 25_000 + uint64(size)*8
		got := p.RequiredGas(input)
		if got != expected {
			t.Errorf("RequiredGas(%d bytes) = %d, want %d", size, got, expected)
		}
	}
}
