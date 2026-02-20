//go:build !proprietary

package pqc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PQCVerifyDecorator is a pass-through stub for public builds.
type PQCVerifyDecorator struct{}

// NewPQCVerifyDecorator creates a new PQC verification ante handler decorator (stub).
func NewPQCVerifyDecorator(_ PQCKeeper, _ PQCClient) PQCVerifyDecorator {
	return PQCVerifyDecorator{}
}

func (PQCVerifyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
