package types

import "cosmossdk.io/errors"

var (
	ErrValidatorNotFound  = errors.Register(ModuleName, 2, "validator reputation not found")
	ErrInvalidParams      = errors.Register(ModuleName, 3, "invalid reputation parameters")
	ErrBelowMinScore      = errors.Register(ModuleName, 4, "validator below minimum reputation score")
)
