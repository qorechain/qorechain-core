//go:build proprietary

package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/ai/types"
)

// EnhancedEngine extends the HeuristicEngine with Phase 2 AI capabilities.
// It wraps the existing AIEngine and adds fraud detection, fee optimization,
// network optimization, and resource allocation.
type EnhancedEngine struct {
	heuristic       *HeuristicEngine      // Phase 1 fast-path (keeps AIEngine interface)
	enhancedRouter  *EnhancedRouter       // Phase 2 enhanced routing
	fraudDetector   *FraudDetector        // Phase 2 fraud detection
	feeOptimizer    *FeeOptimizer         // Phase 2 fee optimization
	networkOptimizer *NetworkOptimizer    // Phase 2 network optimization
	resourceAllocator *ResourceAllocator  // Phase 2 resource allocation
}

// Ensure EnhancedEngine implements AIEngine.
var _ types.AIEngine = (*EnhancedEngine)(nil)

// NewEnhancedEngine creates a new enhanced AI engine with all Phase 2 components.
func NewEnhancedEngine(maxTxPerMinute int, routerConfig types.EnhancedRouterConfig) *EnhancedEngine {
	return &EnhancedEngine{
		heuristic:         NewHeuristicEngine(maxTxPerMinute),
		enhancedRouter:    NewEnhancedRouter(routerConfig),
		fraudDetector:     NewFraudDetector(),
		feeOptimizer:      NewFeeOptimizer(),
		networkOptimizer:  NewNetworkOptimizer(),
		resourceAllocator: NewResourceAllocator(),
	}
}

// RouteTransaction uses the enhanced router (falls back to heuristic).
func (e *EnhancedEngine) RouteTransaction(ctx context.Context, tx types.TransactionInfo) (*types.RoutingDecision, error) {
	return e.enhancedRouter.RouteTransaction(ctx, tx)
}

// DetectAnomaly uses the heuristic anomaly detector.
func (e *EnhancedEngine) DetectAnomaly(ctx context.Context, tx types.TransactionInfo, history []types.TransactionInfo) (*types.AnomalyResult, error) {
	return e.heuristic.DetectAnomaly(ctx, tx, history)
}

// ScoreContractRisk uses the heuristic risk scorer.
func (e *EnhancedEngine) ScoreContractRisk(ctx context.Context, code []byte, chain string) (*types.RiskScore, error) {
	return e.heuristic.ScoreContractRisk(ctx, code, chain)
}

// DetectFraud runs the Phase 2 multi-layered fraud detection.
// blockHeight is used to generate deterministic investigation IDs.
func (e *EnhancedEngine) DetectFraud(ctx context.Context, tx types.TransactionInfo, history []types.TransactionInfo, blockHeight int64) (*types.FraudResult, error) {
	return e.fraudDetector.DetectFraud(ctx, tx, history, blockHeight)
}

// EstimateFee returns a fee estimate for the given urgency.
func (e *EnhancedEngine) EstimateFee(ctx context.Context, urgency string) (*types.FeeEstimate, error) {
	return e.feeOptimizer.EstimateFee(ctx, urgency)
}

// OptimizeNetwork returns network parameter recommendations.
func (e *EnhancedEngine) OptimizeNetwork(ctx context.Context) ([]types.NetworkRecommendation, error) {
	return e.networkOptimizer.Optimize(ctx)
}

// RecommendResources returns resource allocation recommendations.
func (e *EnhancedEngine) RecommendResources(ctx context.Context) (*ResourceAllocation, error) {
	return e.resourceAllocator.Recommend(ctx)
}

// FeeOptimizer returns the fee optimizer for block stats recording.
func (e *EnhancedEngine) FeeOptimizer() *FeeOptimizer {
	return e.feeOptimizer
}

// NetworkOptimizer returns the network optimizer for state recording.
func (e *EnhancedEngine) NetworkOptimizer() *NetworkOptimizer {
	return e.networkOptimizer
}

// EnhancedRouter returns the enhanced router for metrics updates.
func (e *EnhancedEngine) EnhancedRouter() *EnhancedRouter {
	return e.enhancedRouter
}

// ---- Keeper Extension Methods ----

// StoreFraudInvestigation persists a fraud investigation to the KV store.
func (k Keeper) StoreFraudInvestigation(ctx sdk.Context, inv types.FraudInvestigation) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.InvestigationPrefix, []byte(inv.ID)...)
	bz, _ := json.Marshal(inv)
	store.Set(key, bz)
}

// GetFraudInvestigation retrieves a fraud investigation by ID.
func (k Keeper) GetFraudInvestigation(ctx sdk.Context, id string) (*types.FraudInvestigation, error) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.InvestigationPrefix, []byte(id)...)
	bz := store.Get(key)
	if bz == nil {
		return nil, fmt.Errorf("investigation %s not found", id)
	}
	var inv types.FraudInvestigation
	if err := json.Unmarshal(bz, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// StoreFeeSnapshot persists a fee snapshot at the given height.
func (k Keeper) StoreFeeSnapshot(ctx sdk.Context, snap types.FeeSnapshot) {
	store := ctx.KVStore(k.storeKey)
	key := types.HeightKey(types.FeeHistoryPrefix, snap.Height)
	bz, _ := json.Marshal(snap)
	store.Set(key, bz)
}

// StoreNetworkRecommendations persists network recommendations for an epoch.
func (k Keeper) StoreNetworkRecommendations(ctx sdk.Context, epoch int64, recs []types.NetworkRecommendation) {
	store := ctx.KVStore(k.storeKey)
	key := types.HeightKey(types.NetworkRecommendationPrefix, epoch)
	bz, _ := json.Marshal(recs)
	store.Set(key, bz)
}

// SetCircuitBreaker activates a circuit breaker for a contract.
func (k Keeper) SetCircuitBreaker(ctx sdk.Context, cb types.CircuitBreakerState) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.CircuitBreakerPrefix, []byte(cb.ContractAddr)...)
	bz, _ := json.Marshal(cb)
	store.Set(key, bz)
}

// GetCircuitBreaker retrieves a circuit breaker state for a contract.
func (k Keeper) GetCircuitBreaker(ctx sdk.Context, contractAddr string) *types.CircuitBreakerState {
	store := ctx.KVStore(k.storeKey)
	key := append(types.CircuitBreakerPrefix, []byte(contractAddr)...)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var cb types.CircuitBreakerState
	if err := json.Unmarshal(bz, &cb); err != nil {
		return nil
	}
	// Check expiry using block time for determinism
	if !cb.ExpiresAt.IsZero() && ctx.BlockTime().After(cb.ExpiresAt) {
		store.Delete(key)
		return nil
	}
	return &cb
}

// AnalyzeTransactionEnhanced runs the full Phase 2 analysis pipeline.
func (k Keeper) AnalyzeTransactionEnhanced(ctx sdk.Context, tx types.TransactionInfo, history []types.TransactionInfo) (*types.AnomalyResult, *types.FraudResult, error) {
	goCtx := context.Background()

	// Phase 1: anomaly detection
	anomalyResult, err := k.engine.DetectAnomaly(goCtx, tx, history)
	if err != nil {
		return nil, nil, err
	}

	k.IncrementTxsRouted(ctx)

	if anomalyResult.IsAnomalous {
		k.IncrementAnomaliesDetected(ctx)
		k.FlagTransaction(ctx, types.FlaggedTx{
			TxHash:       tx.TxHash,
			AnomalyScore: anomalyResult.Score,
			Flags:        anomalyResult.Flags,
			Height:       ctx.BlockHeight(),
		})
	}

	// Phase 2: fraud detection (if engine supports it)
	var fraudResult *types.FraudResult
	if enhanced, ok := k.engine.(*EnhancedEngine); ok {
		fr, err := enhanced.DetectFraud(goCtx, tx, history, ctx.BlockHeight())
		if err != nil {
			k.logger.Error("fraud detection failed", "error", err)
		} else {
			fraudResult = fr
			if fr.Action != "allow" {
				// Generate deterministic investigation ID from block height + tx hash
				// instead of using time.Now() which breaks consensus.
				invID := fmt.Sprintf("INV-%d-%s", ctx.BlockHeight(), tx.TxHash)
				if len(invID) > 48 {
					invID = invID[:48]
				}
				// Store investigation
				k.StoreFraudInvestigation(ctx, types.FraudInvestigation{
					ID:          invID,
					ThreatType:  fr.ThreatType,
					ThreatLevel: fr.ThreatLevel,
					Sender:      tx.Sender,
					Details:     fr.Details,
					TxHash:      tx.TxHash,
					Height:      ctx.BlockHeight(),
					Timestamp:   ctx.BlockTime(),
				})

				// Circuit break if needed
				if fr.Action == "circuit_break" && tx.Receiver != "" {
					k.SetCircuitBreaker(ctx, types.CircuitBreakerState{
						ContractAddr: tx.Receiver,
						Reason:       fr.Details,
						ActivatedAt:  ctx.BlockTime(),
						ExpiresAt:    ctx.BlockTime().Add(1 * time.Hour),
					})
				}
			}
		}
	}

	return anomalyResult, fraudResult, nil
}
