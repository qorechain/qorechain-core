package types

import "cosmossdk.io/errors"

var (
	ErrInvalidLockAmount   = errors.Register(ModuleName, 2, "invalid lock amount")
	ErrPositionNotFound    = errors.Register(ModuleName, 3, "xQORE position not found")
	ErrInsufficientBalance = errors.Register(ModuleName, 4, "insufficient xQORE balance")
	ErrMinLockAmount       = errors.Register(ModuleName, 5, "amount below minimum lock")
	ErrInvalidParams       = errors.Register(ModuleName, 6, "invalid xQORE params")
)
