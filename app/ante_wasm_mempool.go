package app

import (
	errorsmod "cosmossdk.io/errors"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// wasmMempoolBlockedMsgs are the CosmWasm entry points that let an unprivileged
// account put new code on the chain and bring it to life. The two combined
// messages (store and instantiate, store and migrate) upload code too and were
// missing (QSR-2026-0382 F1, 0396, 0558); code_upload_access=Nobody already
// refuses them at execution, so this only keeps the mempool guard consistent.
var wasmMempoolBlockedMsgs = map[string]struct{}{
	sdk.MsgTypeURL(&wasmtypes.MsgStoreCode{}):                   {},
	sdk.MsgTypeURL(&wasmtypes.MsgInstantiateContract{}):         {},
	sdk.MsgTypeURL(&wasmtypes.MsgInstantiateContract2{}):        {},
	sdk.MsgTypeURL(&wasmtypes.MsgStoreAndInstantiateContract{}): {},
	sdk.MsgTypeURL(&wasmtypes.MsgStoreAndMigrateContract{}):     {},
}

// wasmMempoolInstantiateMsgs are the members of wasmMempoolBlockedMsgs that
// only instantiate code already on the chain. From FeatureWasmGovernedDeploy
// they are admitted: the code they can reach was stored by governance, and
// whether they run is decided in block execution by instantiate permissions and
// the circuit breaker. The upload messages stay refused, since the governance
// path that stores code never passes through a mempool.
var wasmMempoolInstantiateMsgs = map[string]struct{}{
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
//
// governedDeploy reports whether FeatureWasmGovernedDeploy is on; when it is,
// instantiation is admitted and only the upload messages are refused. A nil
// switch is off.
type WasmMempoolGuardDecorator struct {
	governedDeploy func(sdk.Context) bool
}

func NewWasmMempoolGuardDecorator(governedDeploy func(sdk.Context) bool) WasmMempoolGuardDecorator {
	return WasmMempoolGuardDecorator{governedDeploy: governedDeploy}
}

// wasmGovernedDeployFeature reads FeatureWasmGovernedDeploy from the x/pqc
// feature store.
func wasmGovernedDeployFeature(k interface {
	ProtocolFeatureEnabled(ctx sdk.Context, name string) bool
}) func(sdk.Context) bool {
	if k == nil {
		return nil
	}
	return func(ctx sdk.Context) bool {
		return k.ProtocolFeatureEnabled(ctx, pqctypes.FeatureWasmGovernedDeploy)
	}
}

func (d WasmMempoolGuardDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	// Only guard the mempool. Simulation is left alone so that tooling can
	// still estimate gas without being told the message is forbidden.
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	allowInstantiate := d.governedDeploy != nil && d.governedDeploy(ctx)
	if err := rejectWasmMsgs(tx.GetMsgs(), 0, allowInstantiate); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}

// rejectBlockedWasmMsgs is the guard as it stands before
// FeatureWasmGovernedDeploy: upload and instantiation are both refused.
func rejectBlockedWasmMsgs(msgs []sdk.Msg, depth int) error {
	return rejectWasmMsgs(msgs, depth, false)
}

// rejectWasmMsgs walks the message tree, descending into authz MsgExec so the
// guard cannot be sidestepped by wrapping the payload in a grant. depth bounds
// the recursion; authz nesting beyond a couple of levels is not legitimate
// traffic. allowInstantiate admits the instantiate-only messages.
func rejectWasmMsgs(msgs []sdk.Msg, depth int, allowInstantiate bool) error {
	if depth > 4 {
		return errorsmod.Wrap(errortypes.ErrUnauthorized,
			"authz nesting too deep to inspect for CosmWasm upload messages")
	}
	for _, msg := range msgs {
		url := sdk.MsgTypeURL(msg)
		_, blocked := wasmMempoolBlockedMsgs[url]
		if _, inst := wasmMempoolInstantiateMsgs[url]; inst && allowInstantiate {
			blocked = false
		}
		if blocked {
			if allowInstantiate {
				return errorsmod.Wrapf(errortypes.ErrUnauthorized,
					"%s is not accepted by this node: CosmWasm code is stored "+
						"only through governance", url)
			}
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
			if err := rejectWasmMsgs(inner, depth+1, allowInstantiate); err != nil {
				return err
			}
		}
	}
	return nil
}
