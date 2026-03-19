package types

import "cosmossdk.io/errors"

var (
	ErrAlreadyRegistered    = errors.Register(ModuleName, 2, "light node already registered")
	ErrNotRegistered        = errors.Register(ModuleName, 3, "light node not registered")
	ErrInvalidNodeType      = errors.Register(ModuleName, 4, "invalid light node type")
	ErrMaxNodesReached      = errors.Register(ModuleName, 5, "maximum number of light nodes reached")
	ErrInsufficientFee      = errors.Register(ModuleName, 6, "insufficient registration fee")
	ErrInvalidParams        = errors.Register(ModuleName, 7, "invalid lightnode params")
	ErrNotEligibleForReward = errors.Register(ModuleName, 8, "light node not eligible for reward")
	ErrHeartbeatTooEarly    = errors.Register(ModuleName, 9, "heartbeat submitted too early")
	ErrUnauthorized         = errors.Register(ModuleName, 10, "unauthorized")
	ErrInvalidVersion       = errors.Register(ModuleName, 11, "invalid node version")
)
