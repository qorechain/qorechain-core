//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// InitGenesis initializes the multi-layer architecture module state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	// Store parameters
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}

	// Store layers
	for _, layer := range gs.Layers {
		if err := k.setLayer(ctx, layer); err != nil {
			panic(err)
		}
	}

	// Store anchors
	for _, anchor := range gs.Anchors {
		if err := k.setAnchor(ctx, anchor); err != nil {
			panic(err)
		}
		// Set as latest anchor for its layer
		if err := k.setLatestAnchor(ctx, anchor); err != nil {
			panic(err)
		}
	}

	k.logger.Info("multilayer module genesis initialized",
		"layers", len(gs.Layers),
		"anchors", len(gs.Anchors),
	)
}

// ExportGenesis exports the multi-layer architecture module state to genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	layers, _ := k.GetAllLayers(ctx)

	// Convert pointer slices to value slices for genesis serialization
	layerValues := make([]types.LayerConfig, 0, len(layers))
	for _, l := range layers {
		layerValues = append(layerValues, *l)
	}

	// Collect all anchors across all layers
	var anchorValues []types.StateAnchor
	for _, layer := range layers {
		anchors, _ := k.GetAnchors(ctx, layer.LayerID)
		for _, a := range anchors {
			anchorValues = append(anchorValues, *a)
		}
	}

	return &types.GenesisState{
		Params:  k.GetParams(ctx),
		Layers:  layerValues,
		Anchors: anchorValues,
	}
}
