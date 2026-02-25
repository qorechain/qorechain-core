package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidBTCTx       = errorsmod.Register(ModuleName, 2, "invalid BTC transaction")
	ErrStakingNotActive   = errorsmod.Register(ModuleName, 3, "BTC staking is not active")
	ErrInvalidCheckpoint  = errorsmod.Register(ModuleName, 4, "invalid checkpoint")
	ErrEpochNotFound      = errorsmod.Register(ModuleName, 5, "epoch not found")
	ErrInsufficientStake  = errorsmod.Register(ModuleName, 6, "insufficient stake amount")
	ErrPositionNotFound   = errorsmod.Register(ModuleName, 7, "staking position not found")
)
