package app

import sdk "github.com/cosmos/cosmos-sdk/types"

// WasmTxTokenDecorator numbers, during block execution, each transaction
// for the per-transaction share of the block wasm budget, once, so the
// budget never hashes transaction bytes per VM call (wasm_block_budget.go).
// It changes nothing else and costs no gas.
type WasmTxTokenDecorator struct {
	token func(sdk.Context) sdk.Context
}

func NewWasmTxTokenDecorator(token func(sdk.Context) sdk.Context) WasmTxTokenDecorator {
	return WasmTxTokenDecorator{token: token}
}

func (d WasmTxTokenDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if d.token != nil {
		ctx = d.token(ctx)
	}
	return next(ctx, tx, simulate)
}
