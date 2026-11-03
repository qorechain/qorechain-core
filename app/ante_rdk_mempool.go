package app

import (
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	rdktypes "github.com/qorechain/qorechain-core/x/rdk/types"
)

// RDKZKBatchMempoolDecorator refuses into this node's mempool a zk rollup
// batch that block execution refuses after the fee is paid: from v3.2.7
// (FeatureRDKZKVerificationKey) a MsgSubmitBatch of a zk rollup that has no
// usable verification key, or that carries no proof. Before v3.2.7 no rollup
// could register a key, so every such batch paid its fee and failed with code
// 21.
//
// CheckTx only, like StakingLockMempoolDecorator: DeliverTx and simulation are
// untouched, so it changes no consensus rule. The reads run on an infinite gas
// meter, so the gas a transaction consumes in CheckTx is the same as before.
//
// It must never refuse what execution would admit, so it checks only the
// messages that see the committed state unchanged (rdkMempoolChecks); the
// keeper side (MempoolCheckMsg) is off below the switch and leaves every case
// it cannot decide to execution.
type RDKZKBatchMempoolDecorator struct {
	check func(sdk.Context, sdk.Msg) error
}

func NewRDKZKBatchMempoolDecorator(check func(sdk.Context, sdk.Msg) error) RDKZKBatchMempoolDecorator {
	return RDKZKBatchMempoolDecorator{check: check}
}

func (d RDKZKBatchMempoolDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if d.check == nil || !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	free := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	for _, msg := range rdkMempoolChecks(tx.GetMsgs()) {
		if err := d.check(free, msg); err != nil {
			return ctx, err
		}
	}
	return next(ctx, tx, simulate)
}

// rdkMempoolChecks returns the leading MsgSubmitBatch messages of a
// transaction. A batch only reads the rollup's verification key, and no batch
// changes it, so each of them sees the committed key at execution. Everything
// from the first other message on (a MsgSetVerificationKey earlier in the same
// transaction changes the key) is left to execution, as is anything wrapped in
// authz MsgExec.
func rdkMempoolChecks(msgs []sdk.Msg) []sdk.Msg {
	var out []sdk.Msg
	for _, msg := range msgs {
		if _, ok := msg.(*rdktypes.MsgSubmitBatch); !ok {
			return out
		}
		out = append(out, msg)
	}
	return out
}

// rdkMempoolChecker is implemented by the full build's rdk keeper. The public
// build's stub keeps no rollups and does not implement it.
type rdkMempoolChecker interface {
	MempoolCheckMsg(ctx sdk.Context, msg sdk.Msg) error
}

// rdkMempoolGate is the check RDKZKBatchMempoolDecorator runs. The keeper
// answers nil below FeatureRDKZKVerificationKey.
func (app *QoreChainApp) rdkMempoolGate(ctx sdk.Context, msg sdk.Msg) error {
	c, ok := app.RDKKeeper.(rdkMempoolChecker)
	if !ok {
		return nil
	}
	return c.MempoolCheckMsg(ctx, msg)
}
