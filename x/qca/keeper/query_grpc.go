package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/qca/types"
)

// queryServer implements the proto-generated qca Query service.
type queryServer struct {
	keeper Keeper
}

// NewQueryServer returns a QueryServer backed by the qca keeper.
func NewQueryServer(k Keeper) types.QueryServer {
	return queryServer{keeper: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Config(goCtx context.Context, _ *types.QueryConfigRequest) (*types.QueryConfigResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	c := q.keeper.GetConfig(ctx)
	return &types.QueryConfigResponse{
		UseReputationWeighting: c.UseReputationWeighting,
		MinReputationScore:     c.MinReputationScore,

		PoolClassificationInterval: c.PoolConfig.ClassificationInterval,
		PoolWeightRpos:             c.PoolConfig.WeightRPoS,
		PoolWeightDpos:             c.PoolConfig.WeightDPoS,
		PoolMinDelegationDpos:      c.PoolConfig.MinDelegationDPoS,
		PoolRepPercentileRpos:      c.PoolConfig.RepPercentileRPoS,

		BondingAlpha:           c.BondingCurveConfig.Alpha,
		BondingBeta:            c.BondingCurveConfig.Beta,
		BondingPhaseMultiplier: c.BondingCurveConfig.PhaseMultiplier,

		SlashingBaseRate:         c.SlashingConfig.BaseRate,
		SlashingEscalationFactor: c.SlashingConfig.EscalationFactor,
		SlashingMaxPenalty:       c.SlashingConfig.MaxPenalty,
		SlashingDecayHalflife:    c.SlashingConfig.DecayHalflife,

		QdrwEnabled:          c.QDRWConfig.Enabled,
		QdrwXqoreMultiplier:  c.QDRWConfig.XQOREMultiplier,
		QdrwRepMinMultiplier: c.QDRWConfig.RepMinMultiplier,
		QdrwRepMaxMultiplier: c.QDRWConfig.RepMaxMultiplier,
	}, nil
}
