//go:build proprietary

package ai

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/ai/keeper"
	"github.com/qorechain/qorechain-core/x/ai/types"
)

// AIAnomalyDecorator sits in the AnteHandler chain after PQC verification.
// It runs anomaly detection on every transaction using the heuristic engine.
type AIAnomalyDecorator struct {
	aiKeeper keeper.Keeper
}

// NewAIAnomalyDecorator creates a new AI anomaly ante handler decorator.
func NewAIAnomalyDecorator(k keeper.Keeper) AIAnomalyDecorator {
	return AIAnomalyDecorator{aiKeeper: k}
}

func (d AIAnomalyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	config := d.aiKeeper.GetConfig(ctx)

	// Build transaction info from the QoreChain SDK tx
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}

	msgs := feeTx.GetMsgs()
	if len(msgs) == 0 {
		return next(ctx, tx, simulate)
	}

	// For MVP, we analyze the first message's signer
	txInfo := types.TransactionInfo{
		TxType: "transfer",
		Height: ctx.BlockHeight(),
	}

	// Extract sender from first message if possible
	type hasGetSigners interface {
		GetSigners() []sdk.AccAddress
	}
	if m, ok := msgs[0].(hasGetSigners); ok {
		signers := m.GetSigners()
		if len(signers) > 0 {
			txInfo.Sender = signers[0].String()
		}
	}

	// Run anomaly detection with empty history for now
	// History will be populated from indexer in future steps
	result, err := d.aiKeeper.AnalyzeTransaction(ctx, txInfo, nil)
	if err != nil {
		d.aiKeeper.Logger().Error("AI anomaly detection failed", "error", err)
		// Non-fatal: let the transaction through if AI fails
		return next(ctx, tx, simulate)
	}

	// Emit event for indexer
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"ai_anomaly_check",
		sdk.NewAttribute("score", types.FormatFloat64(result.Score)),
		sdk.NewAttribute("action", result.Action),
		sdk.NewAttribute("is_anomalous", types.FormatBool(result.IsAnomalous)),
	))

	// If the anomaly score exceeds the reject threshold, reject the TX
	if result.Action == "reject" && result.Score > config.AnomalyThreshold {
		return ctx, types.ErrTxRejected.Wrapf("anomaly score %.2f exceeds threshold %.2f", result.Score, config.AnomalyThreshold)
	}

	return next(ctx, tx, simulate)
}
