package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrFairBlockDisabled  = errorsmod.Register(ModuleName, 2, "fairblock module is disabled")
	ErrInvalidEncryptedTx = errorsmod.Register(ModuleName, 3, "invalid encrypted transaction")
	ErrDecryptionFailed   = errorsmod.Register(ModuleName, 4, "decryption failed")
	ErrInsufficientShares = errorsmod.Register(ModuleName, 5, "insufficient decryption shares")
	ErrTxTooLarge         = errorsmod.Register(ModuleName, 6, "encrypted transaction too large")
)
