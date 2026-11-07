package app

import (
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	crossvmtypes "github.com/qorechain/qorechain-core/x/crossvm/types"
)

// CrossVMPayloadMempoolDecorator refuses into this node's mempool a
// MsgCrossVMCall whose payload is longer than the crossvm max_message_size
// parameter. Block execution refuses such a call (ErrMessageTooLarge) only
// after the fee is paid; ValidateBasic cannot check it, since it is stateless
// and the limit is a parameter.
//
// CheckTx only, like RDKZKBatchMempoolDecorator: DeliverTx and simulation are
// untouched, so it changes no consensus rule and needs no protocol feature.
// The parameter read runs on an infinite gas meter, so the gas a transaction
// consumes in CheckTx is the same as before.
//
// It must never refuse what execution would admit, so it looks at the
// transaction's own messages and into authz MsgExec (an inner failure fails
// the whole transaction) but not into group proposals (a proposal that fails
// to execute does not fail its submission), and it stops at a crossvm
// MsgUpdateParams, which may change the limit for what follows it.
type CrossVMPayloadMempoolDecorator struct {
	maxPayload func(sdk.Context) uint64
}

func NewCrossVMPayloadMempoolDecorator(maxPayload func(sdk.Context) uint64) CrossVMPayloadMempoolDecorator {
	return CrossVMPayloadMempoolDecorator{maxPayload: maxPayload}
}

// errStopCrossVMWalk ends the walk early without refusing the transaction.
var errStopCrossVMWalk = crossvmtypes.ErrInvalidMessage.Wrap("stop")

func (d CrossVMPayloadMempoolDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if d.maxPayload == nil || !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	var limit uint64
	loaded := false
	err := walkMempoolMsgs(tx.GetMsgs(), "cross-VM calls", false, func(msg sdk.Msg) error {
		switch m := msg.(type) {
		case *crossvmtypes.MsgUpdateParams:
			return errStopCrossVMWalk
		case *crossvmtypes.MsgCrossVMCall:
			if !loaded {
				limit = d.maxPayload(ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()))
				loaded = true
			}
			// A zero limit is not a real configuration (Params.Validate
			// refuses it); leave such a node's calls to execution.
			if limit > 0 && uint64(len(m.Payload)) > limit {
				return crossvmtypes.ErrMessageTooLarge.Wrapf("payload %d exceeds max %d", len(m.Payload), limit)
			}
		}
		return nil
	})
	if err != nil && err != errStopCrossVMWalk { //nolint:errorlint // sentinel identity
		return ctx, err
	}
	return next(ctx, tx, simulate)
}

// crossVMMaxPayload is the limit CrossVMPayloadMempoolDecorator applies: the
// committed crossvm max_message_size parameter.
func (app *QoreChainApp) crossVMMaxPayload(ctx sdk.Context) uint64 {
	if app.CrossVMKeeper == nil {
		return 0
	}
	return app.CrossVMKeeper.GetParams(ctx).MaxMessageSize
}
