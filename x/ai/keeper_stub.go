//go:build !proprietary

package ai

import (
	"context"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// stubAIEngine is a no-op implementation of AIEngine for public builds.
type stubAIEngine struct{}

func (stubAIEngine) RouteTransaction(_ context.Context, _ types.TransactionInfo) (*types.RoutingDecision, error) {
	return &types.RoutingDecision{
		Priority:   0,
		Reason:     "stub routing",
		Confidence: 1.0,
	}, nil
}

func (stubAIEngine) DetectAnomaly(_ context.Context, _ types.TransactionInfo, _ []types.TransactionInfo) (*types.AnomalyResult, error) {
	return &types.AnomalyResult{
		IsAnomalous: false,
		Score:       0.0,
		Action:      "allow",
		Confidence:  1.0,
	}, nil
}

func (stubAIEngine) ScoreContractRisk(_ context.Context, _ []byte, _ string) (*types.RiskScore, error) {
	return &types.RiskScore{
		Score:          0.0,
		Severity:       "LOW",
		Recommendation: "deploy",
	}, nil
}

// StubKeeper is a no-op implementation of AIKeeper for public builds.
type StubKeeper struct {
	engine types.AIEngine
	logger log.Logger
}

// NewStubKeeper creates a new stub AI keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		engine: stubAIEngine{},
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Engine() types.AIEngine                { return k.engine }
func (k *StubKeeper) Logger() log.Logger                     { return k.logger }
func (k *StubKeeper) GetConfig(_ sdk.Context) types.AIConfig { return types.DefaultAIConfig() }
func (k *StubKeeper) SetConfig(_ sdk.Context, _ types.AIConfig) error { return nil }
func (k *StubKeeper) GetStats(_ sdk.Context) types.AIStats   { return types.AIStats{} }
func (k *StubKeeper) SetStats(_ sdk.Context, _ types.AIStats) {}
func (k *StubKeeper) IncrementTxsRouted(_ sdk.Context)        {}
func (k *StubKeeper) IncrementAnomaliesDetected(_ sdk.Context) {}
func (k *StubKeeper) FlagTransaction(_ sdk.Context, _ types.FlaggedTx) {}
func (k *StubKeeper) AnalyzeTransaction(_ sdk.Context, _ types.TransactionInfo, _ []types.TransactionInfo) (*types.AnomalyResult, error) {
	return &types.AnomalyResult{
		IsAnomalous: false,
		Score:       0.0,
		Action:      "allow",
		Confidence:  1.0,
	}, nil
}
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
