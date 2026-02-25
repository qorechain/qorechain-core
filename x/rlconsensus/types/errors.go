package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrRLDisabled           = errorsmod.Register(ModuleName, 2, "RL consensus module is disabled")
	ErrAgentPaused          = errorsmod.Register(ModuleName, 3, "RL agent is paused")
	ErrInvalidPolicyWeights = errorsmod.Register(ModuleName, 4, "invalid policy weights")
	ErrCircuitBreakerActive = errorsmod.Register(ModuleName, 5, "circuit breaker is active")
	ErrInvalidAgentMode     = errorsmod.Register(ModuleName, 6, "invalid agent mode")
	ErrInvalidObservation   = errorsmod.Register(ModuleName, 7, "invalid observation vector")
	ErrOverflow             = errorsmod.Register(ModuleName, 8, "fixed-point arithmetic overflow")
	ErrInvalidRewardWeights = errorsmod.Register(ModuleName, 9, "reward weights must sum to 1.0")
)
