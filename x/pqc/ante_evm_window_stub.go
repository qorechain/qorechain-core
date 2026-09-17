//go:build !full

package pqc

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EVMWindowKeeper is the interface the EVM-lane window decorator needs.
type EVMWindowKeeper interface {
	HasPQCAccount(ctx sdk.Context, address string) bool
	ConsumeEVMWindow(ctx sdk.Context, addr sdk.AccAddress, outflow sdkmath.Int) error
	ProtocolFeatureEnabled(ctx sdk.Context, name string) bool
}

// EVMWindowDecorator is a pass-through stub for public builds.
type EVMWindowDecorator struct{}

// NewEVMWindowDecorator creates the EVM-lane authorisation window decorator (stub).
func NewEVMWindowDecorator(_ EVMWindowKeeper) EVMWindowDecorator {
	return EVMWindowDecorator{}
}

func (EVMWindowDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
