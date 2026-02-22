//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// QueryServer implements the multilayer query handler.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer returns a new QueryServer for the multilayer module.
func NewQueryServer(keeper Keeper) QueryServer {
	return QueryServer{keeper: keeper}
}

// Layer returns a specific layer configuration.
func (qs QueryServer) Layer(ctx sdk.Context, layerID string) (*types.LayerConfig, error) {
	return qs.keeper.GetLayer(ctx, layerID)
}

// Layers returns all registered layers, optionally filtered by type and status.
func (qs QueryServer) Layers(ctx sdk.Context, layerType types.LayerType, status types.LayerStatus) ([]*types.LayerConfig, error) {
	all, err := qs.keeper.GetAllLayers(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*types.LayerConfig
	for _, layer := range all {
		if layerType != "" && layer.LayerType != layerType {
			continue
		}
		if status != "" && layer.Status != status {
			continue
		}
		filtered = append(filtered, layer)
	}
	return filtered, nil
}

// Anchor returns the latest state anchor for a layer.
func (qs QueryServer) Anchor(ctx sdk.Context, layerID string) (*types.StateAnchor, error) {
	return qs.keeper.GetLatestAnchor(ctx, layerID)
}

// Anchors returns all state anchors for a layer.
func (qs QueryServer) Anchors(ctx sdk.Context, layerID string) ([]*types.StateAnchor, error) {
	return qs.keeper.GetAnchors(ctx, layerID)
}

// RoutingStats returns the QCAI routing statistics.
func (qs QueryServer) RoutingStats(ctx sdk.Context) (*types.QueryRoutingStatsResponse, error) {
	return qs.keeper.GetRoutingStats(ctx)
}

// SimulateRoute simulates QCAI routing for a transaction without executing it.
func (qs QueryServer) SimulateRoute(ctx sdk.Context, payload []byte, maxLatency uint64, maxFee string) (*types.RoutingDecision, error) {
	return qs.keeper.SimulateRoute(ctx, payload, maxLatency, maxFee)
}

// Params returns the module parameters.
func (qs QueryServer) Params(ctx sdk.Context) types.Params {
	return qs.keeper.GetParams(ctx)
}
