package app

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// protoTxProvider is implemented by the SDK's decoded tx wrapper.
type protoTxProvider interface {
	GetProtoTx() *txtypes.Tx
}

// refuseFeelessTx refuses a transaction that carries no Fee before anything
// reads it, in CheckTx, ReCheckTx and simulate.
//
// The SDK's tx wrapper (v0.53.6) dereferences AuthInfo.Fee without a nil check,
// and so does the EVM lane's ValidateTx. On the Cosmos lane SetUpContextDecorator
// is the first thing to ask for the gas limit, so such a transaction panics on
// either lane. baseapp recovers the panic, but the recovered error used to carry
// the whole stack trace back to whoever called simulate or broadcast. This turns
// it into a plain error on the public paths, before any panic.
//
// It is called ahead of lane routing so both lanes are covered. Block execution
// keeps its current behaviour (the panic is recovered and the transaction fails
// with the same code as before), so this can be rolled out one node at a time.
// Every well-formed transaction carries a Fee, including Ethereum transactions
// wrapped by MsgEthereumTx.BuildTx, so nothing valid is refused.
func refuseFeelessTx(ctx sdk.Context, tx sdk.Tx, simulate bool) error {
	if !ctx.IsCheckTx() && !simulate {
		return nil
	}
	p, ok := tx.(protoTxProvider)
	if !ok {
		return nil
	}
	ptx := p.GetProtoTx()
	if ptx == nil {
		return nil
	}
	if ptx.AuthInfo == nil || ptx.AuthInfo.Fee == nil {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest, "tx has no fee")
	}
	return nil
}
