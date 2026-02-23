package types

import "fmt"

// AlgorithmID uniquely identifies a PQC algorithm in the agility framework.
// IDs 1-2 are reserved for the initial algorithms; 3-255 are reserved for
// future governance-approved algorithms.
type AlgorithmID uint32

const (
	AlgorithmUnspecified AlgorithmID = 0
	AlgorithmDilithium5 AlgorithmID = 1 // NIST FIPS 204 — signatures
	AlgorithmMLKEM1024  AlgorithmID = 2 // NIST FIPS 203 — key encapsulation
)

// String returns the human-readable name for the algorithm.
func (id AlgorithmID) String() string {
	switch id {
	case AlgorithmUnspecified:
		return "unspecified"
	case AlgorithmDilithium5:
		return "dilithium5"
	case AlgorithmMLKEM1024:
		return "mlkem1024"
	default:
		return fmt.Sprintf("algorithm_%d", id)
	}
}

// IsSignature returns true if this algorithm is a digital signature scheme.
func (id AlgorithmID) IsSignature() bool {
	switch id {
	case AlgorithmDilithium5:
		return true
	default:
		return false
	}
}

// IsKEM returns true if this algorithm is a key encapsulation mechanism.
func (id AlgorithmID) IsKEM() bool {
	switch id {
	case AlgorithmMLKEM1024:
		return true
	default:
		return false
	}
}

// AlgorithmIDFromString parses an algorithm name to its ID.
func AlgorithmIDFromString(name string) (AlgorithmID, error) {
	switch name {
	case "dilithium5", "DILITHIUM5":
		return AlgorithmDilithium5, nil
	case "mlkem1024", "MLKEM1024":
		return AlgorithmMLKEM1024, nil
	default:
		return AlgorithmUnspecified, fmt.Errorf("unknown algorithm: %s", name)
	}
}

// AlgorithmStatus represents the lifecycle state of an algorithm.
type AlgorithmStatus uint32

const (
	StatusActive     AlgorithmStatus = 0 // Fully operational
	StatusMigrating  AlgorithmStatus = 1 // Dual-signature period active
	StatusDeprecated AlgorithmStatus = 2 // Still verifiable, no new key registrations
	StatusDisabled   AlgorithmStatus = 3 // Cannot verify (emergency kill switch)
)

// String returns the human-readable name for the status.
func (s AlgorithmStatus) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusMigrating:
		return "migrating"
	case StatusDeprecated:
		return "deprecated"
	case StatusDisabled:
		return "disabled"
	default:
		return fmt.Sprintf("status_%d", s)
	}
}

// AlgorithmCategory identifies whether an algorithm is for signatures or KEM.
const (
	CategorySignature = "signature"
	CategoryKEM       = "kem"
)

// AlgorithmInfo describes a registered algorithm in the agility framework.
type AlgorithmInfo struct {
	ID             AlgorithmID     `json:"id"`
	Name           string          `json:"name"`            // e.g., "dilithium5"
	Category       string          `json:"category"`        // "signature" or "kem"
	NISTLevel      uint32          `json:"nist_level"`      // Security category (1-5)
	Status         AlgorithmStatus `json:"status"`          // Lifecycle state
	PublicKeySize  uint32          `json:"public_key_size"`
	PrivateKeySize uint32          `json:"private_key_size"`
	SignatureSize  uint32          `json:"signature_size,omitempty"`  // 0 for KEM algorithms
	CiphertextSize uint32         `json:"ciphertext_size,omitempty"` // 0 for signature algorithms
	AddedAtHeight  int64           `json:"added_at_height"`          // Block height when registered
	DeprecatedAt   int64           `json:"deprecated_at,omitempty"`  // 0 if not deprecated
}

// Validate performs basic validation on algorithm info.
func (a AlgorithmInfo) Validate() error {
	if a.ID == AlgorithmUnspecified {
		return ErrInvalidAlgorithm.Wrap("algorithm ID cannot be unspecified")
	}
	if a.Name == "" {
		return ErrInvalidAlgorithm.Wrap("algorithm name is required")
	}
	if a.Category != CategorySignature && a.Category != CategoryKEM {
		return ErrInvalidAlgorithm.Wrapf("invalid category: %s (must be 'signature' or 'kem')", a.Category)
	}
	if a.NISTLevel < 1 || a.NISTLevel > 5 {
		return ErrInvalidAlgorithm.Wrapf("NIST level must be 1-5, got %d", a.NISTLevel)
	}
	if a.PublicKeySize == 0 {
		return ErrInvalidAlgorithm.Wrap("public key size must be > 0")
	}
	if a.PrivateKeySize == 0 {
		return ErrInvalidAlgorithm.Wrap("private key size must be > 0")
	}
	return nil
}

// MigrationInfo tracks the state of an ongoing algorithm migration.
type MigrationInfo struct {
	FromAlgorithmID    AlgorithmID `json:"from_algorithm_id"`
	ToAlgorithmID      AlgorithmID `json:"to_algorithm_id"`
	StartHeight        int64       `json:"start_height"`
	EndHeight          int64       `json:"end_height"`          // StartHeight + MigrationBlocks
	MigratedAccounts   uint64      `json:"migrated_accounts"`   // Counter
	RemainingAccounts  uint64      `json:"remaining_accounts"`  // Counter
}

// DefaultDilithium5Info returns the default AlgorithmInfo for Dilithium-5.
func DefaultDilithium5Info() AlgorithmInfo {
	return AlgorithmInfo{
		ID:             AlgorithmDilithium5,
		Name:           "dilithium5",
		Category:       CategorySignature,
		NISTLevel:      5,
		Status:         StatusActive,
		PublicKeySize:  2592,
		PrivateKeySize: 4896,
		SignatureSize:  4627,
		CiphertextSize: 0,
		AddedAtHeight:  0,
		DeprecatedAt:   0,
	}
}

// DefaultMLKEM1024Info returns the default AlgorithmInfo for ML-KEM-1024.
func DefaultMLKEM1024Info() AlgorithmInfo {
	return AlgorithmInfo{
		ID:             AlgorithmMLKEM1024,
		Name:           "mlkem1024",
		Category:       CategoryKEM,
		NISTLevel:      5,
		Status:         StatusActive,
		PublicKeySize:  1568,
		PrivateKeySize: 3168,
		SignatureSize:  0,
		CiphertextSize: 1568,
		AddedAtHeight:  0,
		DeprecatedAt:   0,
	}
}
