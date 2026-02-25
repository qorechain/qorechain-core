//go:build !proprietary

package pqc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PQCHybridVerifyDecorator is a pass-through stub for public builds.
// In the proprietary build, this verifies PQC hybrid signature TX extensions.
type PQCHybridVerifyDecorator struct{}

// NewPQCHybridVerifyDecorator creates a new hybrid PQC verification ante handler decorator (stub).
func NewPQCHybridVerifyDecorator(_ PQCKeeper, _ PQCClient) PQCHybridVerifyDecorator {
	return PQCHybridVerifyDecorator{}
}

func (PQCHybridVerifyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
