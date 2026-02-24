//go:build !proprietary

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// PQCVerifyPrecompile is the community build stub for PQC signature verification.
// Returns an error indicating this feature requires the full build.
type PQCVerifyPrecompile struct{}

func (p *PQCVerifyPrecompile) Address() common.Address { return PQCVerifyAddress }

func (p *PQCVerifyPrecompile) RequiredGas(input []byte) uint64 {
	return 25_000 + uint64(len(input))*8
}

func (p *PQCVerifyPrecompile) Run(_ *vm.EVM, _ *vm.Contract, _ bool) ([]byte, error) {
	return nil, fmt.Errorf("PQC verification precompile not available in community build")
}
