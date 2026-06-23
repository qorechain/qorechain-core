package amm

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/qorechain/qorechain-core/x/amm/types"
)

// queryServer implements the AMM gRPC Query service over the AMMKeeper interface.
// It maps the keeper's in-memory state types to the query-facing proto views.
type queryServer struct {
	keeper AMMKeeper
}

// NewQueryServer returns an AMM QueryServer backed by the given keeper.
func NewQueryServer(k AMMKeeper) types.QueryServer {
	return queryServer{keeper: k}
}

var _ types.QueryServer = queryServer{}

func poolView(p types.Pool) *types.PoolView {
	return &types.PoolView{
		Id:                       p.ID,
		PoolType:                 string(p.Type),
		Creator:                  p.Creator,
		TokenA:                   p.TokenA,
		TokenB:                   p.TokenB,
		ReserveA:                 p.ReserveA.String(),
		ReserveB:                 p.ReserveB.String(),
		LpSupply:                 p.LPSupply.String(),
		LpDenom:                  p.LPDenom,
		CreatedAt:                p.CreatedAt,
		Status:                   string(p.Status),
		WeightedAvgPrice:         p.WeightedAvgPrice.String(),
		AmplificationCoefficient: p.AmplificationCoefficient,
	}
}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p := q.keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: &types.ParamsView{
		SwapFeeBps:         p.SwapFeeBps,
		ProtocolFeeBps:     p.ProtocolFeeBps,
		MinLiquidity:       p.MinLiquidity.String(),
		MaxPoolsPerCreator: p.MaxPoolsPerCreator,
		LpTokenDecimals:    p.LPTokenDecimals,
		PoolCreationFee:    p.PoolCreationFee.String(),
		MaxSwapImpactBps:   p.MaxSwapImpactBps,
		Enabled:            p.Enabled,
	}}, nil
}

func (q queryServer) Pool(goCtx context.Context, req *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p, ok := q.keeper.GetPool(ctx, req.PoolId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "pool %d not found", req.PoolId)
	}
	return &types.QueryPoolResponse{Pool: poolView(p)}, nil
}

func (q queryServer) Pools(goCtx context.Context, _ *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	pools := q.keeper.GetAllPools(ctx)
	out := make([]*types.PoolView, len(pools))
	for i := range pools {
		out[i] = poolView(pools[i])
	}
	return &types.QueryPoolsResponse{Pools: out}, nil
}

func (q queryServer) PoolByDenoms(goCtx context.Context, req *types.QueryPoolByDenomsRequest) (*types.QueryPoolByDenomsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p, ok := q.keeper.GetPoolByDenoms(ctx, req.DenomA, req.DenomB)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no pool for %s/%s", req.DenomA, req.DenomB)
	}
	return &types.QueryPoolByDenomsResponse{Pool: poolView(p)}, nil
}

func (q queryServer) LPBalance(goCtx context.Context, req *types.QueryLPBalanceRequest) (*types.QueryLPBalanceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	bal := q.keeper.GetLPBalance(ctx, req.PoolId, addr)
	return &types.QueryLPBalanceResponse{Balance: bal.String()}, nil
}

func (q queryServer) QuoteExactIn(goCtx context.Context, req *types.QueryQuoteExactInRequest) (*types.QueryQuoteExactInResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	amtIn, ok := math.NewIntFromString(req.AmountIn)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid amount_in")
	}
	out, fee, err := q.keeper.QuoteExactIn(ctx, req.PoolId, req.DenomIn, amtIn)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &types.QueryQuoteExactInResponse{AmountOut: out.String(), Fee: fee.String()}, nil
}

func (q queryServer) QuoteExactOut(goCtx context.Context, req *types.QueryQuoteExactOutRequest) (*types.QueryQuoteExactOutResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	amtOut, ok := math.NewIntFromString(req.AmountOut)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid amount_out")
	}
	in, fee, err := q.keeper.QuoteExactOut(ctx, req.PoolId, req.DenomOut, amtOut)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &types.QueryQuoteExactOutResponse{AmountIn: in.String(), Fee: fee.String()}, nil
}
