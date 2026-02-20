//go:build !proprietary

package ai

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AIAnomalyDecorator is a pass-through stub for public builds.
type AIAnomalyDecorator struct{}

// NewAIAnomalyDecorator creates a new AI anomaly ante handler decorator (stub).
func NewAIAnomalyDecorator(_ AIKeeper) AIAnomalyDecorator {
	return AIAnomalyDecorator{}
}

func (AIAnomalyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return next(ctx, tx, simulate)
}
