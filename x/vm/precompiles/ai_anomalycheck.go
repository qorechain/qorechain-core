//go:build proprietary

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	aimod "github.com/qorechain/qorechain-core/x/ai"
	aitypes "github.com/qorechain/qorechain-core/x/ai/types"
)

// AIAnomalyCheckPrecompile checks anomaly scores from Solidity.
// Uses ONLY the on-chain deterministic heuristic engine.
type AIAnomalyCheckPrecompile struct {
	aiKeeper aimod.AIKeeper
}

// NewAIAnomalyCheckPrecompile creates a new AI anomaly check precompile instance.
func NewAIAnomalyCheckPrecompile(keeper aimod.AIKeeper) *AIAnomalyCheckPrecompile {
	return &AIAnomalyCheckPrecompile{aiKeeper: keeper}
}

func (p *AIAnomalyCheckPrecompile) Address() common.Address { return AIAnomalyCheckAddress }

func (p *AIAnomalyCheckPrecompile) RequiredGas(_ []byte) uint64 { return 40_000 }

func (p *AIAnomalyCheckPrecompile) Run(evm *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	addr, amount, err := DecodeAIAnomalyCheckInput(contract.Input)
	if err != nil {
		return EncodeAIAnomalyCheckOutput(big.NewInt(0), false)
	}

	ctx, err := getSDKContext(evm)
	if err != nil {
		return EncodeAIAnomalyCheckOutput(big.NewInt(0), false)
	}

	txInfo := aitypes.TransactionInfo{
		Sender: sdk.AccAddress(addr.Bytes()).String(),
		Amount: amount.Uint64(),
		TxType: "transfer",
	}

	result, err := p.aiKeeper.AnalyzeTransaction(ctx, txInfo, nil)
	if err != nil {
		return EncodeAIAnomalyCheckOutput(big.NewInt(0), false)
	}

	// Convert 0.0-1.0 score to 0-10000 basis points
	scoreBP := big.NewInt(int64(result.Score * 10000))
	flagged := result.IsAnomalous

	return EncodeAIAnomalyCheckOutput(scoreBP, flagged)
}
