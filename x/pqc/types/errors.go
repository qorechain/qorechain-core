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
)
