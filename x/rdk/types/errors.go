package types

import "errors"

var (
	ErrRollupNotFound        = errors.New("rdk: rollup not found")
	ErrRollupAlreadyExists   = errors.New("rdk: rollup already exists")
	ErrRollupNotActive       = errors.New("rdk: rollup not active")
	ErrMaxRollupsReached     = errors.New("rdk: maximum rollups reached")
	ErrInsufficientStake     = errors.New("rdk: insufficient stake amount")
	ErrUnauthorized          = errors.New("rdk: unauthorized — only creator can manage rollup")
	ErrBatchNotFound         = errors.New("rdk: batch not found")
	ErrBatchAlreadyFinalized = errors.New("rdk: batch already finalized")
	ErrChallengeWindowClosed = errors.New("rdk: challenge window closed or not applicable")
	ErrInvalidProof          = errors.New("rdk: invalid proof")
	ErrProofRequired         = errors.New("rdk: proof required for this settlement mode")
	ErrDABlobTooLarge        = errors.New("rdk: DA blob exceeds maximum size")
	ErrDABlobNotFound        = errors.New("rdk: DA blob not found")
	ErrCelestiaDAStubed      = errors.New("rdk: Celestia DA backend is stubbed in v1.3.0")
	ErrInvalidSettlementMode = errors.New("rdk: invalid settlement mode")
	ErrInvalidSequencerMode  = errors.New("rdk: invalid sequencer mode")
	ErrInvalidProofSystem    = errors.New("rdk: invalid proof system")
	ErrBasedSequencerOnly    = errors.New("rdk: based settlement requires based sequencer")
	ErrChallengeBondRequired = errors.New("rdk: challenge bond is required")
)
