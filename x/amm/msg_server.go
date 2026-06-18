package amm

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/amm/types"
)

// msgServer implements the proto-generated AMM MsgServer by delegating to the
// AMMKeeper interface (the keeper methods consume the message types directly).
type msgServer struct {
	keeper AMMKeeper
}

// NewMsgServer returns an AMM MsgServer backed by the given keeper.
func NewMsgServer(k AMMKeeper) types.MsgServer {
	return msgServer{keeper: k}
}

var _ types.MsgServer = msgServer{}

func (s msgServer) CreatePool(goCtx context.Context, msg *types.MsgCreatePool) (*types.MsgCreatePoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	pool, _, err := s.keeper.CreatePool(ctx, *msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgCreatePoolResponse{PoolID: pool.ID}, nil
}

func (s msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if _, err := s.keeper.AddLiquidity(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgAddLiquidityResponse{}, nil
}

func (s msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if _, _, err := s.keeper.RemoveLiquidity(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgRemoveLiquidityResponse{}, nil
}

func (s msgServer) SwapExactIn(goCtx context.Context, msg *types.MsgSwapExactIn) (*types.MsgSwapExactInResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if _, err := s.keeper.SwapExactIn(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgSwapExactInResponse{}, nil
}

func (s msgServer) SwapExactOut(goCtx context.Context, msg *types.MsgSwapExactOut) (*types.MsgSwapExactOutResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if _, err := s.keeper.SwapExactOut(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgSwapExactOutResponse{}, nil
}

func (s msgServer) PausePool(goCtx context.Context, msg *types.MsgPausePool) (*types.MsgPausePoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := s.keeper.PausePool(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgPausePoolResponse{}, nil
}

func (s msgServer) ResumePool(goCtx context.Context, msg *types.MsgResumePool) (*types.MsgResumePoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := s.keeper.ResumePool(ctx, *msg); err != nil {
		return nil, err
	}
	return &types.MsgResumePoolResponse{}, nil
}
