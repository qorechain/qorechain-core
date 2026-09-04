package app

import (
	errorsmod "cosmossdk.io/errors"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// wasmMempoolBlockedMsgs are the CosmWasm entry points that let an unprivileged
// account put new code on the chain and bring it to life.
var wasmMempoolBlockedMsgs = map[string]struct{}{
	sdk.MsgTypeURL(&wasmtypes.MsgStoreCode{}):            {},
	sdk.MsgTypeURL(&wasmtypes.MsgInstantiateContract{}):  {},
	sdk.MsgTypeURL(&wasmtypes.MsgInstantiateContract2{}): {},
}

// WasmMempoolGuardDecorator refuses to admit CosmWasm upload/instantiate
// messages into this node's mempool.
//
// It deliberately runs in CheckTx only. DeliverTx is left untouched, so the
// validity rule for a block is unchanged: a block that already contains such a
// transaction still executes identically on every node, patched or not. That
// makes this safe to roll out one node at a time, with no coordinated height
// and no risk of a fork — unlike a DeliverTx rejection, which would be
// consensus-breaking.
//
// The trade-off is that this is block-production policy, not a consensus rule:
// it holds only for as long as the operators of the honest proposers run it. On
// a network where we produce every block it is fully effective; the moment an
// unpatched proposer joins, it is not. It exists to cover the window between
// discovering permissionless wasm upload and a governance proposal setting
// code_upload_access to Nobody, and should be removed once that has executed.
type WasmMempoolGuardDecorator struct{}

func NewWasmMempoolGuardDecorator() WasmMempoolGuardDecorator {
	return WasmMempoolGuardDecorator{}
}

func (d WasmMempoolGuardDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	// Only guard the mempool. Simulation is left alone so that tooling can
	// still estimate gas without being told the message is forbidden.
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	if err := rejectBlockedWasmMsgs(tx.GetMsgs(), 0); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}

// rejectBlockedWasmMsgs walks the message tree, descending into authz MsgExec
// so the guard cannot be sidestepped by wrapping the payload in a grant. depth
// bounds the recursion; authz nesting beyond a couple of levels is not
// legitimate traffic.
func rejectBlockedWasmMsgs(msgs []sdk.Msg, depth int) error {
	if depth > 4 {
		return errorsmod.Wrap(errortypes.ErrUnauthorized,
			"authz nesting too deep to inspect for CosmWasm upload messages")
	}
	for _, msg := range msgs {
		url := sdk.MsgTypeURL(msg)
		if _, blocked := wasmMempoolBlockedMsgs[url]; blocked {
			return errorsmod.Wrapf(errortypes.ErrUnauthorized,
				"%s is not accepted by this node: CosmWasm code upload and "+
					"instantiation are suspended pending a security fix", url)
		}
		if exec, ok := msg.(*authz.MsgExec); ok {
			inner, err := exec.GetMessages()
			if err != nil {
				// An MsgExec whose payload will not unpack cannot be inspected,
				// so it cannot be shown to be safe.
				return errorsmod.Wrap(errortypes.ErrUnauthorized,
					"authz payload could not be unpacked for inspection")
			}
			if err := rejectBlockedWasmMsgs(inner, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
