package types

import "cosmossdk.io/errors"

var (
	ErrChainNotSupported          = errors.Register(ModuleName, 2, "chain not supported")
	ErrChainPaused                = errors.Register(ModuleName, 3, "bridge for chain is paused")
	ErrInvalidAttestation         = errors.Register(ModuleName, 4, "invalid bridge attestation")
	ErrDuplicateAttestation       = errors.Register(ModuleName, 5, "duplicate attestation")
	ErrInsufficientAttestations   = errors.Register(ModuleName, 6, "insufficient attestations for threshold")
	ErrValidatorNotRegistered     = errors.Register(ModuleName, 7, "validator not registered as bridge validator")
	ErrValidatorNotAuthorized     = errors.Register(ModuleName, 8, "validator not authorized for this chain")
	ErrExceedsSingleTransferLimit = errors.Register(ModuleName, 9, "transfer exceeds single transfer limit")
	ErrExceedsDailyLimit          = errors.Register(ModuleName, 10, "transfer exceeds daily limit")
	ErrBridgePaused               = errors.Register(ModuleName, 11, "bridge is paused")
	ErrInvalidPQCSignature        = errors.Register(ModuleName, 12, "invalid PQC signature on attestation")
	ErrOperationNotFound          = errors.Register(ModuleName, 13, "bridge operation not found")
	ErrOperationAlreadyCompleted  = errors.Register(ModuleName, 14, "bridge operation already completed")
	ErrInvalidAmount              = errors.Register(ModuleName, 15, "invalid bridge amount")
	ErrAssetNotSupported          = errors.Register(ModuleName, 16, "asset not supported on this chain")
	ErrInvalidDestination         = errors.Register(ModuleName, 17, "invalid destination address")
	ErrChallengePeriodActive      = errors.Register(ModuleName, 18, "challenge period still active")
)
