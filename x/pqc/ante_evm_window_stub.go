//go:build !full

package pqc

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// EVMWindowKeeper is the interface the EVM-lane window decorator needs.
type EVMWindowKeeper interface {
	HasPQCAccount(ctx sdk.Context, address string) bool
	GetEVMWindow(ctx sdk.Context, addr sdk.AccAddress) (types.EVMAuthWindow, bool)
	ConsumeEVMWindow(ctx sdk.Context, addr sdk.AccAddress, outflow sdkmath.Int) error
	ConsumeEVMDelegationConsent(ctx sdk.Context, addr sdk.AccAddress, delegate common.Address) error
	ProtocolFeatureEnabled(ctx sdk.Context, name string) bool
}

// EVMStateReader is what the decorator reads from the EVM module to tell which
// EIP-7702 authorizations will take effect.
type EVMStateReader interface {
	GetNonce(ctx sdk.Context, addr common.Address) uint64
	GetCodeHash(ctx sdk.Context, addr common.Address) common.Hash
	GetCode(ctx sdk.Context, codeHash common.Hash) []byte
}

// EVMWindowDecorator is a pass-through stub for public builds. It still marks
// the context as an EVM-lane transaction, which is not a check but a fact.
type EVMWindowDecorator struct{}

// NewEVMWindowDecorator creates the EVM-lane authorisation window decorator (stub).
func NewEVMWindowDecorator(_ EVMWindowKeeper, _ EVMStateReader) EVMWindowDecorator {
	return EVMWindowDecorator{}
}

func (EVMWindowDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(types.WithEVMLaneAnte(ctx), tx, simulate)
}
