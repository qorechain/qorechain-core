package types

import "fmt"

// GenesisState defines the multilayer module's genesis state
type GenesisState struct {
	Params  Params        `json:"params"`
	Layers  []LayerConfig `json:"layers"`
	Anchors []StateAnchor `json:"anchors"`
}

// DefaultGenesisState returns the default genesis state for the multi-layer architecture module
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Layers:  []LayerConfig{},
		Anchors: []StateAnchor{},
	}
}

// Validate performs basic validation of the genesis state
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate layer uniqueness
	layerIDs := make(map[string]bool)
	for _, layer := range gs.Layers {
		if layer.LayerID == "" {
			return fmt.Errorf("layer has empty layer_id")
		}
		if layerIDs[layer.LayerID] {
			return fmt.Errorf("duplicate layer_id: %s", layer.LayerID)
		}
		layerIDs[layer.LayerID] = true

		if layer.LayerType != LayerTypeSidechain && layer.LayerType != LayerTypePaychain {
			return fmt.Errorf("layer %s has invalid type: %s", layer.LayerID, layer.LayerType)
		}
	}

	// Validate anchors reference existing layers
	for _, anchor := range gs.Anchors {
		if anchor.LayerID == "" {
			return fmt.Errorf("anchor has empty layer_id")
		}
		if !layerIDs[anchor.LayerID] {
			return fmt.Errorf("anchor references non-existent layer: %s", anchor.LayerID)
		}
		if len(anchor.StateRoot) == 0 {
			return fmt.Errorf("anchor for layer %s at height %d has empty state root", anchor.LayerID, anchor.LayerHeight)
		}
	}

	return nil
}
