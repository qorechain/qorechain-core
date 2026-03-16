//go:build !full

package svm

import (
	"context"

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

// SVMBankKeeper defines the bank keeper methods required by the SVM fee decorator.
type SVMBankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// SVMDeductFeeDecorator is a pass-through stub for public builds.
type SVMDeductFeeDecorator struct{}

func NewSVMDeductFeeDecorator(_ SVMKeeper, _ SVMBankKeeper) SVMDeductFeeDecorator {
	return SVMDeductFeeDecorator{}
}

func (SVMDeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
