//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// QueryServer implements query handlers for the bridge module.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer creates a new bridge query server.
func NewQueryServer(k Keeper) *QueryServer {
	return &QueryServer{keeper: k}
}

// Chains returns all supported chain configurations.
func (q *QueryServer) Chains(ctx sdk.Context) []types.ChainConfig {
	return q.keeper.GetAllChainConfigs(ctx)
}

// Chain returns a specific chain configuration.
func (q *QueryServer) Chain(ctx sdk.Context, chainID string) (types.ChainConfig, bool) {
	return q.keeper.GetChainConfig(ctx, chainID)
}

// Validators returns all bridge validators.
func (q *QueryServer) Validators(ctx sdk.Context) []types.BridgeValidator {
	return q.keeper.GetAllBridgeValidators(ctx)
}

// Operations returns all bridge operations (most recent first).
func (q *QueryServer) Operations(ctx sdk.Context) []types.BridgeOperation {
	return q.keeper.GetAllOperations(ctx)
}

// Operation returns a specific bridge operation.
func (q *QueryServer) Operation(ctx sdk.Context, operationID string) (types.BridgeOperation, bool) {
	return q.keeper.GetOperation(ctx, operationID)
}

// LockedAmount returns the locked/minted amounts for a chain/asset pair.
func (q *QueryServer) LockedAmount(ctx sdk.Context, chain, asset string) types.LockedAmount {
	return q.keeper.GetLockedAmount(ctx, chain, asset)
}

// CircuitBreakers returns all circuit breaker states.
func (q *QueryServer) CircuitBreakers(ctx sdk.Context) []types.CircuitBreakerState {
	return q.keeper.GetAllCircuitBreakers(ctx)
}

// ChainLimits returns circuit breaker limits for a specific chain.
func (q *QueryServer) ChainLimits(ctx sdk.Context, chain string) types.CircuitBreakerState {
	return q.keeper.GetCircuitBreaker(ctx, chain)
}

// BridgeEstimate returns an AI-optimized route estimate.
func (q *QueryServer) BridgeEstimate(ctx sdk.Context, from, to, asset, amount string) (*types.BridgeRouteEstimate, error) {
	optimizer := NewPathOptimizer(q.keeper)
	return optimizer.OptimizePath(ctx, from, to, asset, amount)
}

// Config returns the bridge configuration.
func (q *QueryServer) Config(ctx sdk.Context) types.BridgeConfig {
	return q.keeper.GetConfig(ctx)
}
