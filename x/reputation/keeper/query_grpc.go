package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// queryServer implements the proto-generated reputation Query service.
type queryServer struct {
	keeper Keeper
}

// NewQueryServer returns a QueryServer backed by the reputation keeper.
func NewQueryServer(k Keeper) types.QueryServer {
	return queryServer{keeper: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p := q.keeper.GetParams(ctx)
	return &types.QueryParamsResponse{
		Alpha:    p.Alpha,
		Beta:     p.Beta,
		Gamma:    p.Gamma,
		Delta:    p.Delta,
		Lambda:   p.Lambda,
		MinScore: p.MinScore,
	}, nil
}
