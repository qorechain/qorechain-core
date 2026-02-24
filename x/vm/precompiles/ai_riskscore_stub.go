//go:build !proprietary

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// AIRiskScorePrecompile is the community build stub for AI risk assessment.
// Returns an error indicating this feature requires the full build.
type AIRiskScorePrecompile struct{}

func (p *AIRiskScorePrecompile) Address() common.Address { return AIRiskScoreAddress }

func (p *AIRiskScorePrecompile) RequiredGas(_ []byte) uint64 { return 50_000 }

func (p *AIRiskScorePrecompile) Run(_ *vm.EVM, _ *vm.Contract, _ bool) ([]byte, error) {
	return nil, fmt.Errorf("AI risk score precompile not available in community build")
}
