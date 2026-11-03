package types

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

	// EVMWindows and EVMDelegationConsents are the live EVM-lane
	// authorisations. They are short-lived, but an export taken while one is
	// open must not silently revoke it on import.
	EVMWindows            []GenesisEVMWindow            `json:"evm_windows,omitempty"`
	EVMDelegationConsents []GenesisEVMDelegationConsent `json:"evm_delegation_consents,omitempty"`

	// KEMKeys are the published KEM keys (v3.2.5). Omitted when there are
	// none, so an export from before the registry existed is unchanged.
	KEMKeys []KEMKeyInfo `json:"kem_keys,omitempty"`

	// Multisigs are the registered post-quantum multisig member sets
	// (v3.2.7). Omitted when there are none, so an export from before the
	// registry existed is unchanged.
	Multisigs []PQCMultisigInfo `json:"multisigs,omitempty"`
}

// GenesisEVMWindow is one account's window in a genesis file.
type GenesisEVMWindow struct {
	Address string        `json:"address"`
	Window  EVMAuthWindow `json:"window"`
}

// GenesisEVMDelegationConsent is one account's delegation consent in a
// genesis file.
type GenesisEVMDelegationConsent struct {
	Address string               `json:"address"`
	Consent EVMDelegationConsent `json:"consent"`
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

	if err := validateGenesisAccounts(gs.Accounts); err != nil {
		return err
	}

	seenWindow := map[string]bool{}
	for _, w := range gs.EVMWindows {
		if _, err := sdk.AccAddressFromBech32(w.Address); err != nil {
			return ErrInvalidEVMWindow.Wrapf("genesis window address %q: %v", w.Address, err)
		}
		if seenWindow[w.Address] {
			return ErrInvalidEVMWindow.Wrapf("duplicate genesis window for %s", w.Address)
		}
		seenWindow[w.Address] = true
		if w.Window.MaxValue.IsNil() || w.Window.UsedValue.IsNil() {
			return ErrInvalidEVMWindow.Wrapf("genesis window for %s lacks its value bounds", w.Address)
		}
	}
	seenConsent := map[string]bool{}
	for _, c := range gs.EVMDelegationConsents {
		if _, err := sdk.AccAddressFromBech32(c.Address); err != nil {
			return ErrInvalidEVMDelegation.Wrapf("genesis consent address %q: %v", c.Address, err)
		}
		if seenConsent[c.Address] {
			return ErrInvalidEVMDelegation.Wrapf("duplicate genesis consent for %s", c.Address)
		}
		seenConsent[c.Address] = true
		if _, err := ParseDelegate(c.Consent.Delegate); err != nil {
			return ErrInvalidEVMDelegation.Wrapf("genesis consent for %s: %v", c.Address, err)
		}
	}

	if err := validateGenesisKEMKeys(gs.KEMKeys, gs.Algorithms); err != nil {
		return err
	}
	return validateGenesisMultisigs(gs.Multisigs, gs.Accounts)
}

// validateGenesisMultisigs checks each member set the way registration does,
// one per account, and that no account holds both a set and a single key.
func validateGenesisMultisigs(sets []PQCMultisigInfo, accounts []PQCAccountInfo) error {
	keyed := make(map[string]bool, len(accounts))
	for _, a := range accounts {
		keyed[a.Address] = true
	}
	seen := map[string]bool{}
	for _, m := range sets {
		if err := m.Validate(); err != nil {
			return errors.Wrapf(err, "genesis multisig %s", m.Address)
		}
		if seen[m.Address] {
			return ErrInvalidMultisig.Wrapf("duplicate genesis multisig for %s", m.Address)
		}
		seen[m.Address] = true
		if keyed[m.Address] {
			return ErrMultisigConflict.Wrapf("genesis account %s holds both a PQC key and a multisig member set", m.Address)
		}
	}
	return nil
}

// validateGenesisKEMKeys checks each published KEM key the way registration
// does: one per account, a key-encapsulation algorithm of the genesis
// registry, that algorithm's key size, and the stateless key rule.
func validateGenesisKEMKeys(keys []KEMKeyInfo, algos []AlgorithmInfo) error {
	byID := make(map[AlgorithmID]AlgorithmInfo, len(algos))
	for _, a := range algos {
		byID[a.ID] = a
	}
	seen := map[string]bool{}
	for _, k := range keys {
		if _, err := sdk.AccAddressFromBech32(k.Address); err != nil {
			return ErrInvalidKEMKey.Wrapf("genesis KEM key address %q: %v", k.Address, err)
		}
		if seen[k.Address] {
			return ErrInvalidKEMKey.Wrapf("duplicate genesis KEM key for %s", k.Address)
		}
		seen[k.Address] = true
		algo, ok := byID[k.AlgorithmID]
		if !ok || algo.Category != CategoryKEM {
			return ErrInvalidAlgorithm.Wrapf("genesis KEM key for %s names %s, not a key-encapsulation algorithm of the genesis registry", k.Address, k.AlgorithmID)
		}
		if uint32(len(k.PublicKey)) != algo.PublicKeySize {
			return ErrInvalidKEMKey.Wrapf("genesis KEM key for %s is %d bytes, %s uses %d", k.Address, len(k.PublicKey), algo.Name, algo.PublicKeySize)
		}
		if err := ValidateKEMPublicKey(k.AlgorithmID, k.PublicKey); err != nil {
			return errors.Wrapf(err, "genesis KEM key for %s", k.Address)
		}
	}
	return nil
}
