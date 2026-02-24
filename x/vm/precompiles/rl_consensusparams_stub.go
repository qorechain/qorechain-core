//go:build !proprietary

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// RLConsensusParamsPrecompile is the community build stub for RL consensus parameters.
// Returns an error indicating this feature requires the full build.
type RLConsensusParamsPrecompile struct{}

func (p *RLConsensusParamsPrecompile) Address() common.Address { return RLConsensusParamsAddress }

func (p *RLConsensusParamsPrecompile) RequiredGas(_ []byte) uint64 { return 1_500 }

func (p *RLConsensusParamsPrecompile) Run(_ *vm.EVM, _ *vm.Contract, _ bool) ([]byte, error) {
	return nil, fmt.Errorf("RL consensus params precompile not available in community build")
}
