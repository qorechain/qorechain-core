package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrProgramNotFound       = errorsmod.Register(ModuleName, 2, "program not found")
	ErrAccountNotFound       = errorsmod.Register(ModuleName, 3, "SVM account not found")
	ErrInvalidBytecode       = errorsmod.Register(ModuleName, 4, "invalid BPF bytecode")
	ErrComputeBudgetExceeded = errorsmod.Register(ModuleName, 5, "compute budget exceeded")
	ErrInsufficientLamports  = errorsmod.Register(ModuleName, 6, "insufficient lamports")
	ErrRentNotExempt         = errorsmod.Register(ModuleName, 7, "account not rent-exempt")
	ErrAccountAlreadyExists  = errorsmod.Register(ModuleName, 8, "SVM account already exists")
	ErrInvalidAccountOwner   = errorsmod.Register(ModuleName, 9, "invalid account owner")
	ErrMaxCPIDepthExceeded   = errorsmod.Register(ModuleName, 10, "max CPI depth exceeded")
	ErrSVMDisabled           = errorsmod.Register(ModuleName, 11, "SVM module is disabled")
	ErrInvalidAddress        = errorsmod.Register(ModuleName, 12, "invalid SVM address")
	ErrProgramNotExecutable  = errorsmod.Register(ModuleName, 13, "program account is not executable")
	ErrInvalidSignature      = errorsmod.Register(ModuleName, 14, "invalid signature")
	ErrInvalidInstruction    = errorsmod.Register(ModuleName, 15, "invalid instruction data")
	ErrCPINotSupported       = errorsmod.Register(ModuleName, 16, "CPI to non-native programs not supported")
	ErrInvalidTokenAccount   = errorsmod.Register(ModuleName, 17, "invalid token account data")
	ErrInvalidMint           = errorsmod.Register(ModuleName, 18, "invalid mint data")
	ErrAirdropExceeded       = errorsmod.Register(ModuleName, 19, "airdrop rate limit exceeded")
	ErrBlockhashExpired      = errorsmod.Register(ModuleName, 20, "blockhash expired")
	ErrNativeProgramError    = errorsmod.Register(ModuleName, 21, "native program execution error")
)
