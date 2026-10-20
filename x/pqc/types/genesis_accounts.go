package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// validateGenesisAccounts checks the PQC key records a genesis imports
// (QSR-2026-0354). InitGenesis stores them as given, so a record that names an
// invalid address, repeats one, carries a key-encapsulation algorithm or a key
// of the wrong size would otherwise lock that account from the first block.
//
// It accepts what a running chain can hold, so an export of a live network
// always imports: a classical_only record with no key (registrations allowed
// it before v3.2.3) and a degenerate key (a chain only refuses new ones).
func validateGenesisAccounts(accounts []PQCAccountInfo) error {
	seen := make(map[string]bool, len(accounts))
	for i, a := range accounts {
		if _, err := sdk.AccAddressFromBech32(a.Address); err != nil {
			return ErrInvalidKeyLength.Wrapf("genesis PQC account %d: address %q: %v", i, a.Address, err)
		}
		if seen[a.Address] {
			return ErrAccountAlreadyExists.Wrapf("duplicate genesis PQC account %s", a.Address)
		}
		seen[a.Address] = true

		switch a.KeyType {
		case KeyTypeHybrid, KeyTypePQCOnly:
			if len(a.PublicKey) == 0 {
				return ErrInvalidKeyLength.Wrapf("genesis PQC account %s: key_type %s needs a public key", a.Address, a.KeyType)
			}
		case KeyTypeClassicalOnly:
		default:
			return ErrInvalidKeyLength.Wrapf("genesis PQC account %s: invalid key_type %q", a.Address, a.KeyType)
		}

		if a.AlgorithmID.IsKEM() {
			return ErrInvalidAlgorithm.Wrapf("genesis PQC account %s: %s is a key-encapsulation algorithm and cannot authenticate", a.Address, a.AlgorithmID)
		}
		if want, known := ExpectedPublicKeySize(a.AlgorithmID); known && len(a.PublicKey) > 0 && len(a.PublicKey) != int(want) {
			return ErrInvalidKeyLength.Wrapf("genesis PQC account %s: a %s public key is %d bytes, got %d", a.Address, a.AlgorithmID, want, len(a.PublicKey))
		}
		if a.MigrationAlgorithmID.IsKEM() {
			return ErrInvalidAlgorithm.Wrapf("genesis PQC account %s: migration to key-encapsulation algorithm %s", a.Address, a.MigrationAlgorithmID)
		}
		for _, h := range a.RetiredKeyHashes {
			if len(h) != 32 {
				return fmt.Errorf("genesis PQC account %s: retired key hash of %d bytes, want 32", a.Address, len(h))
			}
		}
	}
	return nil
}
