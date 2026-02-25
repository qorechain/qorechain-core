package types

import "cosmossdk.io/errors"

var (
	ErrPQCVerifyFailed     = errors.Register(ModuleName, 2, "PQC signature verification failed")
	ErrInvalidKeyLength    = errors.Register(ModuleName, 3, "invalid PQC key length")
	ErrInvalidSigLength    = errors.Register(ModuleName, 4, "invalid PQC signature length")
	ErrKeygenFailed        = errors.Register(ModuleName, 5, "PQC key generation failed")
	ErrAccountNotFound     = errors.Register(ModuleName, 6, "PQC account not found")
	ErrAccountAlreadyExists = errors.Register(ModuleName, 7, "PQC account already registered")
	ErrClassicalFallback   = errors.Register(ModuleName, 8, "classical fallback not allowed")
	ErrFFICallFailed       = errors.Register(ModuleName, 9, "FFI call to libqorepqc failed")

	// Algorithm agility errors (v0.6.0)
	ErrInvalidAlgorithm       = errors.Register(ModuleName, 10, "invalid or unsupported PQC algorithm")
	ErrAlgorithmNotActive     = errors.Register(ModuleName, 11, "PQC algorithm is not active")
	ErrAlgorithmAlreadyExists = errors.Register(ModuleName, 12, "PQC algorithm already registered")
	ErrAlgorithmDisabled      = errors.Register(ModuleName, 13, "PQC algorithm has been disabled")
	ErrMigrationActive        = errors.Register(ModuleName, 14, "algorithm migration already in progress")
	ErrMigrationNotActive     = errors.Register(ModuleName, 15, "no active migration for this algorithm")
	ErrDualSigRequired        = errors.Register(ModuleName, 16, "dual signature required during migration")
	ErrDualSigInvalid         = errors.Register(ModuleName, 17, "dual signature verification failed")
	ErrKeyMigrationFailed     = errors.Register(ModuleName, 18, "PQC key migration failed")
	ErrUnauthorizedGovAction  = errors.Register(ModuleName, 19, "unauthorized governance action")

	// Hybrid signature errors (v1.1.0)
	ErrHybridSigRequired  = errors.Register(ModuleName, 20, "hybrid PQC signature required")
	ErrHybridSigInvalid   = errors.Register(ModuleName, 21, "hybrid PQC signature verification failed")
	ErrHybridModeDisabled = errors.Register(ModuleName, 22, "hybrid signature mode is disabled")
	ErrInvalidHybridSig   = errors.Register(ModuleName, 23, "invalid hybrid signature format")
)
