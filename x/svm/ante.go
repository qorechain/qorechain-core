//go:build proprietary

package svm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/qorechain/qorechain-core/x/svm/types"
)

// SVMComputeBudgetDecorator validates that SVM execution messages request
// a compute budget within the allowed maximum.
type SVMComputeBudgetDecorator struct {
	svmKeeper SVMKeeper
}

func NewSVMComputeBudgetDecorator(svmKeeper SVMKeeper) SVMComputeBudgetDecorator {
	return SVMComputeBudgetDecorator{svmKeeper: svmKeeper}
}

func (d SVMComputeBudgetDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs := tx.GetMsgs()

	// Lazy-load params only once per transaction, and only if SVM messages exist.
	var params types.Params
	paramsLoaded := false

	for _, msg := range msgs {
		switch m := msg.(type) {
		case *types.MsgExecuteProgram:
			if !paramsLoaded {
				params = d.svmKeeper.GetParams(ctx)
				paramsLoaded = true
			}
			if !params.Enabled {
				return ctx, types.ErrSVMDisabled
			}
			// MsgExecuteProgram doesn't carry a compute budget field yet;
			// the runtime enforces params.ComputeBudgetMax during execution.
			// This decorator validates the SVM module is enabled.
			_ = m

		case *types.MsgDeployProgram:
			if !paramsLoaded {
				params = d.svmKeeper.GetParams(ctx)
				paramsLoaded = true
			}
			if !params.Enabled {
				return ctx, types.ErrSVMDisabled
			}
			if uint64(len(m.Bytecode)) > params.MaxProgramSize {
				return ctx, types.ErrInvalidBytecode.Wrapf(
					"program size %d exceeds max %d", len(m.Bytecode), params.MaxProgramSize)
			}

		case *types.MsgCreateAccount:
			if !paramsLoaded {
				params = d.svmKeeper.GetParams(ctx)
				paramsLoaded = true
			}
			if !params.Enabled {
				return ctx, types.ErrSVMDisabled
			}
		}
	}
	return next(ctx, tx, simulate)
}

// SVMDeductFeeDecorator deducts SVM-specific fees from the sender for
// program deployment and execution messages.
type SVMDeductFeeDecorator struct {
	svmKeeper SVMKeeper
}

func NewSVMDeductFeeDecorator(svmKeeper SVMKeeper) SVMDeductFeeDecorator {
	return SVMDeductFeeDecorator{svmKeeper: svmKeeper}
}

func (d SVMDeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// SVM fee deduction is handled at the keeper level during execution.
	// This decorator is a placeholder for future SVM-specific fee logic
	// (e.g., upfront compute unit reservation, priority fees).
	//
	// Standard SDK fee deduction via DeductFeeDecorator already covers
	// the gas costs for the wrapping transaction.
	return next(ctx, tx, simulate)
}
