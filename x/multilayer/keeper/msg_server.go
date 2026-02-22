//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// MsgServer implements the multilayer message handler.
// All message handling is delegated to the Keeper methods.
type MsgServer struct {
	keeper Keeper
}

// NewMsgServer returns a new MsgServer for the multilayer module.
func NewMsgServer(keeper Keeper) MsgServer {
	return MsgServer{keeper: keeper}
}

// RegisterSidechain handles MsgRegisterSidechain.
func (ms MsgServer) RegisterSidechain(ctx sdk.Context, msg *types.MsgRegisterSidechain) (*types.MsgRegisterSidechainResponse, error) {
	return ms.keeper.RegisterSidechain(ctx, msg)
}

// RegisterPaychain handles MsgRegisterPaychain.
func (ms MsgServer) RegisterPaychain(ctx sdk.Context, msg *types.MsgRegisterPaychain) (*types.MsgRegisterPaychainResponse, error) {
	return ms.keeper.RegisterPaychain(ctx, msg)
}

// AnchorState handles MsgAnchorState.
func (ms MsgServer) AnchorState(ctx sdk.Context, msg *types.MsgAnchorState) (*types.MsgAnchorStateResponse, error) {
	return ms.keeper.AnchorState(ctx, msg)
}

// RouteTransaction handles MsgRouteTransaction.
func (ms MsgServer) RouteTransaction(ctx sdk.Context, msg *types.MsgRouteTransaction) (*types.MsgRouteTransactionResponse, error) {
	return ms.keeper.RouteTransaction(ctx, msg)
}

// UpdateLayerStatus handles MsgUpdateLayerStatus.
func (ms MsgServer) UpdateLayerStatus(ctx sdk.Context, msg *types.MsgUpdateLayerStatus) (*types.MsgUpdateLayerStatusResponse, error) {
	err := ms.keeper.UpdateLayerStatus(ctx, msg.LayerID, msg.NewStatus, msg.Reason)
	if err != nil {
		return nil, err
	}
	return &types.MsgUpdateLayerStatusResponse{}, nil
}

// ChallengeAnchor handles MsgChallengeAnchor.
func (ms MsgServer) ChallengeAnchor(ctx sdk.Context, msg *types.MsgChallengeAnchor) (*types.MsgChallengeAnchorResponse, error) {
	return ms.keeper.ChallengeAnchor(ctx, msg)
}

// UpdateParams handles MsgUpdateParams.
func (ms MsgServer) UpdateParams(ctx sdk.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := ms.keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
