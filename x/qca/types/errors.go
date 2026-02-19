package types

import "cosmossdk.io/errors"

var (
	ErrNoValidators    = errors.Register(ModuleName, 2, "no active validators available")
	ErrInvalidConfig   = errors.Register(ModuleName, 3, "invalid QCA configuration")
)
