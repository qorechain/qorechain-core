//go:build proprietary

package keeper

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// PathOptimizer implements AI-driven bridge path selection.
// When multiple bridge routes exist, the optimizer selects the optimal path
// based on cost, time, and security metrics.
type PathOptimizer struct {
	keeper Keeper
}

// NewPathOptimizer creates a new AI-driven path optimizer.
func NewPathOptimizer(k Keeper) *PathOptimizer {
	return &PathOptimizer{keeper: k}
}

// PathOption represents a possible bridge route.
type PathOption struct {
	Route          []string // Chain IDs in order
	EstimatedFee   string
	EstimatedTime  int64   // seconds
	SecurityScore  float64 // 0.0 to 1.0
	Confidence     float64
}

// OptimizePath finds the best bridge route between two chains.
// Uses the whitepaper's optimization formula adapted for bridge routing:
// OptimalPath = argmin_p(α·Time(p) + β·Cost(p) + γ·Risk(p))
func (o *PathOptimizer) OptimizePath(ctx sdk.Context, sourceChain, destChain, asset, amount string) (*types.BridgeRouteEstimate, error) {
	// Weights for path optimization
	const (
		alphaTime     = 0.4
		betaCost      = 0.3
		gammaRisk     = 0.3
	)

	// Get available routes
	routes := o.findRoutes(ctx, sourceChain, destChain, asset)
	if len(routes) == 0 {
		// Direct route
		directEstimate := o.estimateDirectRoute(ctx, sourceChain, destChain, asset, amount)
		return directEstimate, nil
	}

	// Score each route
	var bestRoute *types.BridgeRouteEstimate
	bestScore := math.MaxFloat64

	for _, route := range routes {
		timeScore := float64(route.EstimatedTime) / 3600.0 // Normalize to hours
		costScore := 0.01 // Flat cost estimate for testnet
		riskScore := 1.0 - route.SecurityScore

		totalScore := alphaTime*timeScore + betaCost*costScore + gammaRisk*riskScore

		if totalScore < bestScore {
			bestScore = totalScore
			bestRoute = &types.BridgeRouteEstimate{
				SourceChain:   sourceChain,
				DestChain:     destChain,
				Asset:         asset,
				Amount:        amount,
				EstimatedFee:  route.EstimatedFee,
				EstimatedTime: route.EstimatedTime,
				Route:         route.Route,
				Confidence:    route.Confidence,
				SecurityScore: route.SecurityScore,
			}
		}
	}

	if bestRoute == nil {
		return o.estimateDirectRoute(ctx, sourceChain, destChain, asset, amount), nil
	}

	return bestRoute, nil
}

// findRoutes discovers available bridge routes between chains.
func (o *PathOptimizer) findRoutes(ctx sdk.Context, sourceChain, destChain, _ string) []PathOption {
	var routes []PathOption

	// Direct route is always an option
	srcConfig, srcFound := o.keeper.GetChainConfig(ctx, sourceChain)
	dstConfig, dstFound := o.keeper.GetChainConfig(ctx, destChain)

	if srcFound && dstFound && srcConfig.Status == types.BridgeStatusActive && dstConfig.Status == types.BridgeStatusActive {
		routes = append(routes, PathOption{
			Route:         []string{sourceChain, destChain},
			EstimatedFee:  "1000", // 1000 uqor flat fee for testnet
			EstimatedTime: 300,    // 5 minutes default
			SecurityScore: 0.9,
			Confidence:    0.85,
		})
	}

	// Multi-hop routes via QoreChain hub
	// E.g., Ethereum → QoreChain → Solana
	if sourceChain != "qorechain" && destChain != "qorechain" {
		if srcFound && srcConfig.Status == types.BridgeStatusActive {
			if dstFound && dstConfig.Status == types.BridgeStatusActive {
				routes = append(routes, PathOption{
					Route:         []string{sourceChain, "qorechain", destChain},
					EstimatedFee:  "2000", // Higher fee for multi-hop
					EstimatedTime: 600,    // ~10 minutes for multi-hop
					SecurityScore: 0.95,   // Higher security via QoreChain hub
					Confidence:    0.8,
				})
			}
		}
	}

	return routes
}

// estimateDirectRoute creates a default estimate for a direct bridge route.
func (o *PathOptimizer) estimateDirectRoute(_ sdk.Context, sourceChain, destChain, asset, amount string) *types.BridgeRouteEstimate {
	return &types.BridgeRouteEstimate{
		SourceChain:   sourceChain,
		DestChain:     destChain,
		Asset:         asset,
		Amount:        amount,
		EstimatedFee:  "1000",
		EstimatedTime: 300,
		Route:         []string{sourceChain, destChain},
		Confidence:    0.7,
		SecurityScore: 0.85,
	}
}
