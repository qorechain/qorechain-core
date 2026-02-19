package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

type msgServer struct {
	keeper Keeper
}

// NewMsgServer returns an implementation of the MsgServer interface.
func NewMsgServer(keeper Keeper) *msgServer {
	return &msgServer{keeper: keeper}
}

// RegisterPQCKey handles the MsgRegisterPQCKey message.
func (s *msgServer) RegisterPQCKey(goCtx context.Context, msg *types.MsgRegisterPQCKey) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if s.keeper.HasPQCAccount(ctx, msg.Sender) {
		return types.ErrAccountAlreadyExists
	}

	info := types.PQCAccountInfo{
		Address:         msg.Sender,
		DilithiumPubkey: msg.DilithiumPubkey,
		ECDSAPubkey:     msg.ECDSAPubkey,
		KeyType:         msg.KeyType,
		CreatedAtHeight: ctx.BlockHeight(),
	}

	if err := s.keeper.SetPQCAccount(ctx, info); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"pqc_key_registered",
		sdk.NewAttribute("address", msg.Sender),
		sdk.NewAttribute("key_type", msg.KeyType),
	))

	return nil
}
