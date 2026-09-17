package types

import "fmt"

// GenesisState defines the pqc module's genesis state.
type GenesisState struct {
	Params   Params           `json:"params"`
	Accounts []PQCAccountInfo `json:"accounts"`
	Stats    PQCStats         `json:"stats"`

	// Algorithm agility genesis (v0.6.0)
	Algorithms []AlgorithmInfo `json:"algorithms"`
	Migrations []MigrationInfo `json:"migrations,omitempty"`

	// ProtocolFeatures are the behaviour switches enabled from the first
	// block. A chain born on this binary lists every released feature; the
	// genuine genesis of a network that upgrades into it lists none, so
	// InitGenesis writes no feature record at height 0 (a store key the
	// network's history never had) and the release upgrade handler enables
	// them at the plan height instead.
	ProtocolFeatures []string `json:"protocol_features,omitempty"`
}

// DefaultGenesisState returns the default genesis state.
// Registers Dilithium-5 and ML-KEM-1024 as the default active algorithms.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Accounts: []PQCAccountInfo{},
		Stats:    PQCStats{},
		Algorithms: []AlgorithmInfo{
			DefaultDilithium5Info(),
			DefaultMLKEM1024Info(),
		},
		Migrations: []MigrationInfo{},
		// Copied, so a caller mutating the genesis cannot edit the release list.
		ProtocolFeatures: append([]string(nil), AllProtocolFeatures...),
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	seenFeature := map[string]bool{}
	for _, f := range gs.ProtocolFeatures {
		if !KnownProtocolFeature(f) {
			return fmt.Errorf("unknown protocol feature %q in genesis", f)
		}
		if seenFeature[f] {
			return fmt.Errorf("duplicate protocol feature %q in genesis", f)
		}
		seenFeature[f] = true
	}
	if gs.Params.MinSecurityLevel < 1 || gs.Params.MinSecurityLevel > 5 {
		return ErrInvalidKeyLength.Wrap("min_security_level must be between 1 and 5")
	}

	// Validate hybrid signature mode (v1.1.0)
	if !gs.Params.HybridSignatureMode.IsValid() {
		return ErrInvalidHybridSig.Wrapf(
			"invalid hybrid_signature_mode %d in genesis params (must be 0, 1, or 2)",
			gs.Params.HybridSignatureMode,
		)
	}

	// Validate algorithms
	seenIDs := make(map[AlgorithmID]bool)
	for _, algo := range gs.Algorithms {
		if err := algo.Validate(); err != nil {
			return err
		}
		if seenIDs[algo.ID] {
			return ErrAlgorithmAlreadyExists.Wrapf("duplicate algorithm ID %d in genesis", algo.ID)
		}
		seenIDs[algo.ID] = true
	}

	// Validate migration references
	for _, mig := range gs.Migrations {
		if !seenIDs[mig.FromAlgorithmID] {
			return ErrInvalidAlgorithm.Wrapf("migration references unknown source algorithm %d", mig.FromAlgorithmID)
		}
		if !seenIDs[mig.ToAlgorithmID] {
			return ErrInvalidAlgorithm.Wrapf("migration references unknown target algorithm %d", mig.ToAlgorithmID)
		}
	}

	return nil
}
