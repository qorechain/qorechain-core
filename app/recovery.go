package app

import (
	"fmt"
	"runtime/debug"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// newQuietPanicRecoveryHandler replaces the SDK's last-resort runTx recovery for
// the response only.
//
// The SDK default puts debug.Stack() into the error text, and that text is what a
// simulate or broadcast caller gets back. It disclosed build paths, the source
// layout and dependency versions to anyone who could make a transaction panic.
// This handler writes the stack to the node's own log and returns the same error
// code (ErrPanic) with only the panic value.
//
// It changes no deterministic result field. The code, gas wanted and gas used are
// the ones the default produced, and the out-of-gas middleware still runs first.
// Only the Log text is shorter, and Log is not part of the results hash (it
// already differed between nodes, because the stack carries goroutine IDs).
func newQuietPanicRecoveryHandler(logger log.Logger) baseapp.RecoveryHandler {
	return func(recoveryObj any) error {
		logger.Error("panic recovered in runTx",
			"panic", fmt.Sprintf("%v", recoveryObj),
			"stack", string(debug.Stack()))
		return errorsmod.Wrap(sdkerrors.ErrPanic, fmt.Sprintf("recovered: %v", recoveryObj))
	}
}
