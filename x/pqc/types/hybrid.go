package types

import "fmt"

// HybridSignatureMode controls chain-wide hybrid signature enforcement.
// Governance-controllable via x/pqc params.
type HybridSignatureMode uint32

const (
	// HybridDisabled means only classical signatures are accepted.
	// PQC extensions on transactions are ignored.
	HybridDisabled HybridSignatureMode = 0

	// HybridOptional means PQC extensions are verified if present.
	// Accounts with registered PQC keys must include the extension.
	// Accounts without PQC keys may transact classically.
	// This is the default mode.
	HybridOptional HybridSignatureMode = 1

	// HybridRequired means all transactions must carry both
	// classical and PQC signatures. Transactions without a PQC
	// extension are rejected. Future governance upgrade path.
	HybridRequired HybridSignatureMode = 2
)

// String returns the human-readable name for the hybrid signature mode.
func (m HybridSignatureMode) String() string {
	switch m {
	case HybridDisabled:
		return "disabled"
	case HybridOptional:
		return "optional"
	case HybridRequired:
		return "required"
	default:
		return fmt.Sprintf("unknown_%d", m)
	}
}

// Description returns a longer description of the hybrid signature mode.
func (m HybridSignatureMode) Description() string {
	switch m {
	case HybridDisabled:
		return "Classical signatures only; PQC extensions are ignored"
	case HybridOptional:
		return "PQC extensions verified if present; classical fallback allowed for accounts without PQC keys"
	case HybridRequired:
		return "Both classical and PQC signatures required on every transaction"
	default:
		return fmt.Sprintf("Unknown hybrid signature mode: %d", m)
	}
}

// IsValid returns true if the mode is a recognized value (0, 1, or 2).
func (m HybridSignatureMode) IsValid() bool {
	return m <= HybridRequired
}

// HybridSignatureModeFromString parses a mode name to its enum value.
func HybridSignatureModeFromString(name string) (HybridSignatureMode, error) {
	switch name {
	case "disabled", "DISABLED", "0":
		return HybridDisabled, nil
	case "optional", "OPTIONAL", "1":
		return HybridOptional, nil
	case "required", "REQUIRED", "2":
		return HybridRequired, nil
	default:
		return HybridDisabled, fmt.Errorf("unknown hybrid signature mode: %s (valid: disabled, optional, required)", name)
	}
}

// HybridSigTypeURL is the type URL for PQCHybridSignature when used as a TX extension.
const HybridSigTypeURL = "/qorechain.pqc.v1.PQCHybridSignature"

// PQCHybridSignature is now a generated protobuf message (see hybrid.pb.go),
// registered as a cosmos.tx.v1beta1.TxExtensionOptionI so it can be carried in
// TxBody.extension_options and resolved by the tx decoder. The helper methods
// below (Validate, HasPublicKey) augment the generated type; Reset/String/
// Marshal/Unmarshal are generated.

// Validate performs basic validation on the hybrid signature.
func (sig PQCHybridSignature) Validate() error {
	if sig.AlgorithmID == AlgorithmUnspecified {
		return ErrInvalidHybridSig.Wrap("algorithm ID cannot be unspecified")
	}

	if !sig.AlgorithmID.IsSignature() {
		return ErrInvalidHybridSig.Wrapf("algorithm %s is not a signature algorithm", sig.AlgorithmID)
	}

	if len(sig.PQCSignature) == 0 {
		return ErrInvalidHybridSig.Wrap("PQC signature cannot be empty")
	}

	// Validate signature size for known algorithms
	switch sig.AlgorithmID {
	case AlgorithmDilithium5:
		if len(sig.PQCSignature) != 4627 {
			return ErrInvalidHybridSig.Wrapf("dilithium5 signature must be 4627 bytes, got %d", len(sig.PQCSignature))
		}
		if len(sig.PQCPublicKey) > 0 && len(sig.PQCPublicKey) != 2592 {
			return ErrInvalidHybridSig.Wrapf("dilithium5 public key must be 2592 bytes, got %d", len(sig.PQCPublicKey))
		}
	}

	return nil
}

// HasPublicKey returns true if the signature includes a PQC public key
// for auto-registration.
func (sig PQCHybridSignature) HasPublicKey() bool {
	return len(sig.PQCPublicKey) > 0
}
