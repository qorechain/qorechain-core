//go:build proprietary

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// RLConsensusParamsPrecompile returns current RL-tuned consensus parameters.
// Falls back to static genesis params if the RL module is not active.
type RLConsensusParamsPrecompile struct {
	provider RLConsensusParamsProvider
}

// NewRLConsensusParamsPrecompile creates a new RL consensus params precompile instance.
func NewRLConsensusParamsPrecompile(provider RLConsensusParamsProvider) *RLConsensusParamsPrecompile {
	return &RLConsensusParamsPrecompile{provider: provider}
}

func (p *RLConsensusParamsPrecompile) Address() common.Address { return RLConsensusParamsAddress }

func (p *RLConsensusParamsPrecompile) RequiredGas(_ []byte) uint64 { return 1_500 }

func (p *RLConsensusParamsPrecompile) Run(evm *vm.EVM, _ *vm.Contract, _ bool) ([]byte, error) {
	ctx, err := getSDKContext(evm)
	if err != nil {
		// Return static defaults if context unavailable
		return EncodeRLConsensusParamsOutput(
			big.NewInt(5000),
			big.NewInt(100),
			big.NewInt(100),
			big.NewInt(0),
		)
	}

	blockTimeMs := p.provider.GetCurrentBlockTime(ctx).Milliseconds()
	baseGasPrice := p.provider.GetCurrentBaseGasPrice(ctx).TruncateInt().BigInt()
	valSetSize := p.provider.GetValidatorSetSize(ctx)
	epoch := p.provider.GetCurrentEpoch(ctx)

	return EncodeRLConsensusParamsOutput(
		big.NewInt(blockTimeMs),
		baseGasPrice,
		new(big.Int).SetUint64(valSetSize),
		new(big.Int).SetUint64(epoch),
	)
}
