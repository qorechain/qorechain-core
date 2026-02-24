//go:build !proprietary

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// CrossVMBridgePrecompile is the community build stub for cross-VM bridge calls.
// Returns an error indicating this feature requires the full build.
type CrossVMBridgePrecompile struct{}

func (p *CrossVMBridgePrecompile) Address() common.Address { return CrossVMBridgeAddress }

func (p *CrossVMBridgePrecompile) RequiredGas(_ []byte) uint64 { return 50_000 }

func (p *CrossVMBridgePrecompile) Run(_ *vm.EVM, _ *vm.Contract, _ bool) ([]byte, error) {
	return nil, fmt.Errorf("cross-VM bridge precompile not available in community build")
}
