package app

import (
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// StakingLockMempoolDecorator refuses into this node's mempool an undelegation
// or redelegation that would take locked stake out of a delegation. Block
// execution refuses such a message with stakingtier code 13 (ErrLockedStake),
// but only after the fee is paid; refusing it at CheckTx spares the wallet a
// fee for a certain failure.
//
// CheckTx only, like WasmMempoolGuardDecorator: DeliverTx and simulation are
// untouched, so no consensus rule changes and it needs no protocol feature.
// The reads run on an infinite gas meter, so the gas a transaction consumes in
// CheckTx is the same as before.
//
// It must never refuse what execution would admit. The keeper side
// (MempoolCheckStakingMsg) allows for the block time and the exchange rate
// moving before execution; this side checks only the messages that execute
// against the committed state unchanged (stakingLockMempoolChecks). A
// transaction that depends on another pending one (MsgUnlockEarly sent
// separately, not yet in a block) is refused until that one is included;
// bundling the MsgUnlockEarly into the same transaction avoids it.
type StakingLockMempoolDecorator struct {
	check func(sdk.Context, sdk.Msg) error
}

func NewStakingLockMempoolDecorator(check func(sdk.Context, sdk.Msg) error) StakingLockMempoolDecorator {
	return StakingLockMempoolDecorator{check: check}
}

func (d StakingLockMempoolDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if d.check == nil || !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	free := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	for _, msg := range stakingLockMempoolChecks(tx.GetMsgs()) {
		if err := d.check(free, msg); err != nil {
			return ctx, err
		}
	}
	return next(ctx, tx, simulate)
}

// stakingLockMempoolChecks returns the messages of a transaction whose lock
// check sees, at execution, the same delegation, validator and locks as the
// committed state: top-level MsgUndelegate and MsgBeginRedelegate, in order,
// for as long as only such messages precede them, and only while no earlier
// one touched their source validator (an earlier message on the same
// validator changes the delegation, or for a redelegation's destination adds
// to it). Everything after any other message type is left to execution, as is
// anything wrapped in authz MsgExec.
func stakingLockMempoolChecks(msgs []sdk.Msg) []sdk.Msg {
	var out []sdk.Msg
	touched := map[string]struct{}{}
	for _, msg := range msgs {
		var src, dst string
		switch m := msg.(type) {
		case *stakingtypes.MsgUndelegate:
			src = m.ValidatorAddress
		case *stakingtypes.MsgBeginRedelegate:
			src, dst = m.ValidatorSrcAddress, m.ValidatorDstAddress
		default:
			return out
		}
		if _, seen := touched[src]; !seen {
			out = append(out, msg)
		}
		touched[src] = struct{}{}
		if dst != "" {
			touched[dst] = struct{}{}
		}
	}
	return out
}

// stakingTierMempoolGate is the check StakingLockMempoolDecorator runs: off
// below the staking tiers switch, like the execution gate.
func (app *QoreChainApp) stakingTierMempoolGate(ctx sdk.Context, msg sdk.Msg) error {
	if app.StakingTierKeeper == nil || !app.stakingTiersActive(ctx) {
		return nil
	}
	return app.StakingTierKeeper.MempoolCheckStakingMsg(ctx, msg)
}
