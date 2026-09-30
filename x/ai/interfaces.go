package ai

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// AIKeeper is the interface for the x/ai module's keeper.
// Used by the ante handler and the app wiring layer.
type AIKeeper interface {
	Engine() types.AIEngine
	Logger() log.Logger

	GetConfig(ctx sdk.Context) types.AIConfig
	SetConfig(ctx sdk.Context, cfg types.AIConfig) error
	GetStats(ctx sdk.Context) types.AIStats
	SetStats(ctx sdk.Context, stats types.AIStats)
	IncrementTxsRouted(ctx sdk.Context)
	IncrementAnomaliesDetected(ctx sdk.Context)
	IncrementContractsScored(ctx sdk.Context)
	IncrementTxsFlagged(ctx sdk.Context)
	IncrementTxsRejected(ctx sdk.Context)
	FlagTransaction(ctx sdk.Context, flagged types.FlaggedTx) error
	AnalyzeTransaction(ctx sdk.Context, tx types.TransactionInfo, history []types.TransactionInfo) (*types.AnomalyResult, error)
	ScoreContract(ctx sdk.Context, code []byte, chain string) (*types.RiskScore, error)

	// PruneLegacyTxHistory deletes the per-height transaction history the
	// screening kept before v3.2.2 and returns how many entries it removed.
	PruneLegacyTxHistory(ctx sdk.Context) int

	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
