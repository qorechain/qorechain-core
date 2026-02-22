package types

import "cosmossdk.io/errors"

// x/multilayer module sentinel errors
var (
	ErrLayerAlreadyExists     = errors.Register(ModuleName, 2, "layer already exists")
	ErrLayerNotFound          = errors.Register(ModuleName, 3, "layer not found")
	ErrLayerNotActive         = errors.Register(ModuleName, 4, "layer is not active")
	ErrMaxSidechainsReached   = errors.Register(ModuleName, 5, "maximum sidechains reached")
	ErrMaxPaychainsReached    = errors.Register(ModuleName, 6, "maximum paychains reached")
	ErrInsufficientStake      = errors.Register(ModuleName, 7, "insufficient stake to create layer")
	ErrInvalidAnchor          = errors.Register(ModuleName, 8, "invalid state anchor")
	ErrAnchorTooFrequent      = errors.Register(ModuleName, 9, "anchor submitted too frequently")
	ErrAnchorTooStale         = errors.Register(ModuleName, 10, "anchor exceeds max interval, force anchor required")
	ErrChallengePeriodExpired = errors.Register(ModuleName, 11, "challenge period has expired")
	ErrInvalidFraudProof      = errors.Register(ModuleName, 12, "invalid fraud proof")
	ErrRoutingDisabled        = errors.Register(ModuleName, 13, "QCAI routing is disabled")
	ErrRoutingLowConfidence   = errors.Register(ModuleName, 14, "QCAI routing confidence below threshold")
	ErrInvalidLayerTransition = errors.Register(ModuleName, 15, "invalid layer status transition")
	ErrUnauthorized           = errors.Register(ModuleName, 16, "unauthorized: insufficient permissions")
	ErrInvalidPQCSignature    = errors.Register(ModuleName, 17, "invalid PQC aggregate signature on anchor")
)
