package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidMessage    = errorsmod.Register(ModuleName, 2, "invalid cross-VM message")
	ErrMessageNotFound   = errorsmod.Register(ModuleName, 3, "cross-VM message not found")
	ErrQueueFull         = errorsmod.Register(ModuleName, 4, "cross-VM message queue is full")
	ErrInvalidTarget     = errorsmod.Register(ModuleName, 5, "invalid target contract address")
	ErrExecutionFailed   = errorsmod.Register(ModuleName, 6, "cross-VM execution failed")
	ErrUnsupportedVM     = errorsmod.Register(ModuleName, 7, "unsupported VM type")
	ErrMessageTooLarge   = errorsmod.Register(ModuleName, 8, "message payload exceeds max size")
	ErrQueueTimeout      = errorsmod.Register(ModuleName, 9, "queued message timed out")
	ErrUnauthorized      = errorsmod.Register(ModuleName, 10, "unauthorized cross-VM call")
	ErrPrecompileCall    = errorsmod.Register(ModuleName, 11, "precompile call failed")
	ErrABIEncoding       = errorsmod.Register(ModuleName, 12, "ABI encoding/decoding failed")
	ErrWasmExecution     = errorsmod.Register(ModuleName, 13, "CosmWasm contract execution failed")
	ErrEVMExecution      = errorsmod.Register(ModuleName, 14, "EVM contract execution failed")
)
