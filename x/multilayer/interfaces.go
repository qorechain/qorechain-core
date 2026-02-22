package multilayer

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// MultilayerKeeper defines the interface for the multilayer module keeper.
// This interface is implemented by the proprietary keeper (real logic)
// and the stub keeper (no-op for public builds).
type MultilayerKeeper interface {
	// Layer Registry
	RegisterSidechain(ctx sdk.Context, msg *types.MsgRegisterSidechain) (*types.MsgRegisterSidechainResponse, error)
	RegisterPaychain(ctx sdk.Context, msg *types.MsgRegisterPaychain) (*types.MsgRegisterPaychainResponse, error)
	GetLayer(ctx sdk.Context, layerID string) (*types.LayerConfig, error)
	GetAllLayers(ctx sdk.Context) ([]*types.LayerConfig, error)
	GetLayersByType(ctx sdk.Context, layerType types.LayerType) ([]*types.LayerConfig, error)
	UpdateLayerStatus(ctx sdk.Context, layerID string, status types.LayerStatus, reason string) error

	// State Anchoring (HCS)
	AnchorState(ctx sdk.Context, msg *types.MsgAnchorState) (*types.MsgAnchorStateResponse, error)
	GetLatestAnchor(ctx sdk.Context, layerID string) (*types.StateAnchor, error)
	GetAnchors(ctx sdk.Context, layerID string) ([]*types.StateAnchor, error)
	ChallengeAnchor(ctx sdk.Context, msg *types.MsgChallengeAnchor) (*types.MsgChallengeAnchorResponse, error)

	// QCAI Transaction Routing
	RouteTransaction(ctx sdk.Context, msg *types.MsgRouteTransaction) (*types.MsgRouteTransactionResponse, error)
	SimulateRoute(ctx sdk.Context, payload []byte, maxLatency uint64, maxFee string) (*types.RoutingDecision, error)
	GetRoutingStats(ctx sdk.Context) (*types.QueryRoutingStatsResponse, error)

	// Cross-Layer Fee Bundling (CLFB)
	CalculateCrossLayerFee(ctx sdk.Context, sourceLayers []string, totalGas uint64) (sdk.Coins, error)

	// Parameters
	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error

	// Genesis
	InitGenesis(ctx sdk.Context, state types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}

// QCAIRouterInterface defines the interface for QCAI-powered transaction routing.
// This is the extension point where heuristic routing will be swapped for
// trained ML models. CRITICAL for future ML upgrade path.
type QCAIRouterInterface interface {
	// ScoreLayers evaluates all active layers for a given transaction.
	// Returns scores per layer based on congestion, capability match, cost.
	ScoreLayers(ctx sdk.Context, txPayload []byte, activeLayers []*types.LayerConfig) ([]*types.LayerScore, error)

	// SelectOptimalLayer picks the best layer based on scores and constraints
	SelectOptimalLayer(scores []*types.LayerScore, maxLatency uint64, maxFee string) (*types.RoutingDecision, error)

	// UpdateRoutingModel feeds back routing outcomes for model improvement.
	// No-op in heuristic mode; critical for reinforcement learning mode.
	UpdateRoutingModel(ctx sdk.Context, decision *types.RoutingDecision, outcome RoutingOutcome) error
}

// RoutingOutcome captures the result of a routed transaction for feedback
type RoutingOutcome struct {
	TransactionHash string
	SelectedLayer   string
	ActualLatencyMs uint64
	ActualGasUsed   uint64
	Success         bool
	ErrorMessage    string
}

// PQCVerifier defines the interface for verifying PQC signatures on state anchors.
// Implemented by x/pqc module.
type PQCVerifier interface {
	VerifyDilithiumSignature(ctx sdk.Context, pubKey []byte, message []byte, signature []byte) (bool, error)
	VerifyAggregateSignature(ctx sdk.Context, pubKeys [][]byte, message []byte, aggregateSig []byte) (bool, error)
}
