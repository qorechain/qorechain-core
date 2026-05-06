package types

import "cosmossdk.io/errors"

const codespace = ModuleName

var (
	ErrPoolNotFound          = errors.Register(codespace, 1, "pool not found")
	ErrInvalidSwap           = errors.Register(codespace, 2, "invalid swap")
	ErrInsufficientLiquidity = errors.Register(codespace, 3, "insufficient liquidity")
	ErrSlippageExceeded      = errors.Register(codespace, 4, "slippage tolerance exceeded")
	ErrPoolPaused            = errors.Register(codespace, 5, "pool is paused")
	ErrInvalidDenoms         = errors.Register(codespace, 6, "invalid token denoms")
	ErrPoolAlreadyExists     = errors.Register(codespace, 7, "pool already exists for this token pair")
	ErrInvalidPoolType       = errors.Register(codespace, 8, "invalid pool type")
	ErrBelowMinLiquidity     = errors.Register(codespace, 9, "deposit below minimum liquidity")
	ErrInvalidLPAmount       = errors.Register(codespace, 10, "invalid LP token amount")
	ErrUnauthorized          = errors.Register(codespace, 11, "unauthorized")
	ErrInvalidParams         = errors.Register(codespace, 12, "invalid module params")
	ErrInvalidPool           = errors.Register(codespace, 13, "invalid pool state")
	ErrNotImplemented        = errors.Register(codespace, 14, "not implemented in community-edition build")
	ErrInvalidAmount         = errors.Register(codespace, 15, "invalid amount")
	ErrSameDenom             = errors.Register(codespace, 16, "tokenA and tokenB must differ")
	ErrPoolLimitReached      = errors.Register(codespace, 17, "pool creator has reached max pools")
)
