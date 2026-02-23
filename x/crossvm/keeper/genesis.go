//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// InitGenesis initializes the crossvm module's state from a genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}
	for _, msg := range gs.Messages {
		if err := k.storeMessage(ctx, msg); err != nil {
			panic(err)
		}
		if msg.Status == types.StatusPending {
			_ = k.enqueueMessage(ctx, msg.ID)
		}
	}
}

// ExportGenesis exports the crossvm module's current state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)

	// Export all stored messages
	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator(
		types.MessageStoreKey(""),
		types.MessageStoreKey("\xff"),
	)
	defer iter.Close()

	var messages []types.CrossVMMessage
	for ; iter.Valid(); iter.Next() {
		msg, err := types.UnmarshalCrossVMMessage(iter.Value())
		if err != nil {
			k.logger.Error("failed to unmarshal message during export", "error", err)
			continue
		}
		messages = append(messages, *msg)
	}

	return &types.GenesisState{
		Params:   params,
		Messages: messages,
	}
}
