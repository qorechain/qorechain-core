package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrRollupNotFound        = errorsmod.Register(ModuleName, 2, "rollup not found")
	ErrRollupAlreadyExists   = errorsmod.Register(ModuleName, 3, "rollup already exists")
	ErrRollupNotActive       = errorsmod.Register(ModuleName, 4, "rollup not active")
	ErrMaxRollupsReached     = errorsmod.Register(ModuleName, 5, "maximum rollups reached")
	ErrInsufficientStake     = errorsmod.Register(ModuleName, 6, "insufficient stake amount")
	ErrUnauthorized          = errorsmod.Register(ModuleName, 7, "unauthorized — only creator can manage rollup")
	ErrBatchNotFound         = errorsmod.Register(ModuleName, 8, "batch not found")
	ErrBatchAlreadyFinalized = errorsmod.Register(ModuleName, 9, "batch already finalized")
	ErrChallengeWindowClosed = errorsmod.Register(ModuleName, 10, "challenge window closed or not applicable")
	ErrInvalidProof          = errorsmod.Register(ModuleName, 11, "invalid proof")
	ErrProofRequired         = errorsmod.Register(ModuleName, 12, "proof required for this settlement mode")
	ErrDABlobTooLarge        = errorsmod.Register(ModuleName, 13, "DA blob exceeds maximum size")
	ErrDABlobNotFound        = errorsmod.Register(ModuleName, 14, "DA blob not found")
	ErrCelestiaDAStubed      = errorsmod.Register(ModuleName, 15, "Celestia DA backend is stubbed in v1.3.0")
	ErrInvalidSettlementMode = errorsmod.Register(ModuleName, 16, "invalid settlement mode")
	ErrInvalidSequencerMode  = errorsmod.Register(ModuleName, 17, "invalid sequencer mode")
	ErrInvalidProofSystem    = errorsmod.Register(ModuleName, 18, "invalid proof system")
	ErrBasedSequencerOnly    = errorsmod.Register(ModuleName, 19, "based settlement requires based sequencer")
	ErrChallengeBondRequired = errorsmod.Register(ModuleName, 20, "challenge bond is required")
)
