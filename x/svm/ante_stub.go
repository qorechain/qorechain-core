//go:build !proprietary

package svm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SVMComputeBudgetDecorator is a pass-through stub for public builds.
type SVMComputeBudgetDecorator struct{}

func NewSVMComputeBudgetDecorator(_ SVMKeeper) SVMComputeBudgetDecorator {
	return SVMComputeBudgetDecorator{}
}

func (SVMComputeBudgetDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}

// SVMDeductFeeDecorator is a pass-through stub for public builds.
type SVMDeductFeeDecorator struct{}

func NewSVMDeductFeeDecorator(_ SVMKeeper) SVMDeductFeeDecorator {
	return SVMDeductFeeDecorator{}
}

func (SVMDeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
