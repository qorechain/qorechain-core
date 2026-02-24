//go:build !proprietary

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// PQCKeyStatusPrecompile is the community build stub for PQC key status queries.
// Returns an error indicating this feature requires the full build.
type PQCKeyStatusPrecompile struct{}

func (p *PQCKeyStatusPrecompile) Address() common.Address { return PQCKeyStatusAddress }

func (p *PQCKeyStatusPrecompile) RequiredGas(_ []byte) uint64 { return 2_500 }

func (p *PQCKeyStatusPrecompile) Run(_ *vm.EVM, _ *vm.Contract, _ bool) ([]byte, error) {
	return nil, fmt.Errorf("PQC key status precompile not available in community build")
}
