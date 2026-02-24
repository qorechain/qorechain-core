//go:build !proprietary

package svm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SVMAnteDecorator is a pass-through stub for public builds.
type SVMAnteDecorator struct{}

// NewSVMAnteDecorator creates an SVM ante handler decorator (stub).
func NewSVMAnteDecorator(_ SVMKeeper) SVMAnteDecorator {
	return SVMAnteDecorator{}
}

func (SVMAnteDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
