package types

import "cosmossdk.io/errors"

var (
	ErrInvalidBurnAmount = errors.Register(ModuleName, 2, "invalid burn amount")
	ErrBurnDisabled      = errors.Register(ModuleName, 3, "burns are disabled")
	ErrInvalidBurnSource = errors.Register(ModuleName, 4, "invalid burn source")
	ErrInvalidParams     = errors.Register(ModuleName, 5, "invalid burn params")
	ErrInsufficientFunds = errors.Register(ModuleName, 6, "insufficient funds for burn")
)
