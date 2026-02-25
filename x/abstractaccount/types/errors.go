package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrAccountNotFound      = errorsmod.Register(ModuleName, 2, "abstract account not found")
	ErrAccountExists        = errorsmod.Register(ModuleName, 3, "abstract account already exists")
	ErrInvalidAccountType   = errorsmod.Register(ModuleName, 4, "invalid account type")
	ErrSpendingLimitExceeded = errorsmod.Register(ModuleName, 5, "spending limit exceeded")
	ErrSessionKeyExpired    = errorsmod.Register(ModuleName, 6, "session key expired")
	ErrMaxSessionKeys       = errorsmod.Register(ModuleName, 7, "maximum session keys reached")
	ErrInvalidSpendingRule  = errorsmod.Register(ModuleName, 8, "invalid spending rule")
	ErrModuleDisabled       = errorsmod.Register(ModuleName, 9, "abstract account module is disabled")
)
