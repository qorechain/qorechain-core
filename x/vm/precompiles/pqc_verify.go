//go:build proprietary

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// PQCVerifyPrecompile verifies ML-DSA (Dilithium-5) signatures from Solidity.
// Allows EVM contracts to verify post-quantum signatures on-chain.
type PQCVerifyPrecompile struct {
	pqcKeeper pqcmod.PQCKeeper
}

// NewPQCVerifyPrecompile creates a new PQC verify precompile instance.
func NewPQCVerifyPrecompile(keeper pqcmod.PQCKeeper) *PQCVerifyPrecompile {
	return &PQCVerifyPrecompile{pqcKeeper: keeper}
}

func (p *PQCVerifyPrecompile) Address() common.Address { return PQCVerifyAddress }

func (p *PQCVerifyPrecompile) RequiredGas(input []byte) uint64 {
	return 25_000 + uint64(len(input))*8
}

func (p *PQCVerifyPrecompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	pubkey, sig, msg, err := DecodePQCVerifyInput(contract.Input)
	if err != nil {
		return EncodePQCVerifyOutput(false)
	}

	// Use the algorithm-aware Verify via PQCClient.
	// Default to Dilithium-5 (AlgorithmID=1) for EVM precompile calls.
	client := p.pqcKeeper.PQCClient()
	valid, err := client.Verify(types.AlgorithmID(1), pubkey, msg, sig)
	if err != nil {
		return EncodePQCVerifyOutput(false)
	}

	return EncodePQCVerifyOutput(valid)
}
