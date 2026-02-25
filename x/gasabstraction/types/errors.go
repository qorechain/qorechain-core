package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrUnsupportedDenom  = errorsmod.Register(ModuleName, 2, "unsupported fee denom")
	ErrConversionFailed  = errorsmod.Register(ModuleName, 3, "fee conversion failed")
	ErrInsufficientFee   = errorsmod.Register(ModuleName, 4, "insufficient fee amount")
	ErrGasAbstractionOff = errorsmod.Register(ModuleName, 5, "gas abstraction disabled")
)
