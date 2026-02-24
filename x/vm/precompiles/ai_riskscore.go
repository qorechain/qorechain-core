//go:build proprietary

package precompiles

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	aimod "github.com/qorechain/qorechain-core/x/ai"
)

// AIRiskScorePrecompile performs deterministic AI risk assessment from Solidity.
// CRITICAL: Uses ONLY the on-chain heuristic engine (Z-score + isolation forest).
// NEVER calls the QCAI Backend sidecar (non-deterministic).
type AIRiskScorePrecompile struct {
	aiKeeper aimod.AIKeeper
}

// NewAIRiskScorePrecompile creates a new AI risk score precompile instance.
func NewAIRiskScorePrecompile(keeper aimod.AIKeeper) *AIRiskScorePrecompile {
	return &AIRiskScorePrecompile{aiKeeper: keeper}
}

func (p *AIRiskScorePrecompile) Address() common.Address { return AIRiskScoreAddress }

func (p *AIRiskScorePrecompile) RequiredGas(_ []byte) uint64 { return 50_000 }

func (p *AIRiskScorePrecompile) Run(_ *vm.EVM, contract *vm.Contract, _ bool) ([]byte, error) {
	txData, err := DecodeAIRiskScoreInput(contract.Input)
	if err != nil {
		// Safe default: score=0, level=SAFE
		return EncodeAIRiskScoreOutput(big.NewInt(0), 0)
	}

	engine := p.aiKeeper.Engine()
	riskScore, err := engine.ScoreContractRisk(context.Background(), txData, "evm")
	if err != nil {
		return EncodeAIRiskScoreOutput(big.NewInt(0), 0)
	}

	// Convert 0.0-1.0 float to 0-10000 basis points
	scoreBP := big.NewInt(int64(riskScore.Score * 10000))
	level := severityToLevel(riskScore.Severity)

	return EncodeAIRiskScoreOutput(scoreBP, level)
}

func severityToLevel(severity string) uint8 {
	switch severity {
	case "LOW":
		return 1
	case "MEDIUM":
		return 2
	case "HIGH":
		return 3
	case "CRITICAL":
		return 4
	default:
		return 0 // SAFE
	}
}
