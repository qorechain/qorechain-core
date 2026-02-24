//go:build !proprietary

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// AIAnomalyCheckPrecompile is the community build stub for anomaly detection.
// Returns an error indicating this feature requires the full build.
type AIAnomalyCheckPrecompile struct{}

func (p *AIAnomalyCheckPrecompile) Address() common.Address { return AIAnomalyCheckAddress }

func (p *AIAnomalyCheckPrecompile) RequiredGas(_ []byte) uint64 { return 40_000 }

func (p *AIAnomalyCheckPrecompile) Run(_ *vm.EVM, _ *vm.Contract, _ bool) ([]byte, error) {
	return nil, fmt.Errorf("AI anomaly check precompile not available in community build")
}
