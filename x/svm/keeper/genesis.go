//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// InitGenesis initializes the SVM module's state from a genesis state.
// It stores parameters, creates all genesis accounts (including built-in
// system programs), and imports program metadata.
func (k *Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	// Store params.
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}

	// Import accounts (system programs and any pre-existing accounts).
	for i := range gs.Accounts {
		acc := &gs.Accounts[i]
		if err := k.SetAccount(ctx, acc); err != nil {
			panic(err)
		}
	}

	// Import program metadata.
	for _, meta := range gs.Programs {
		if err := k.SetProgramMeta(ctx, meta); err != nil {
			panic(err)
		}
	}

	// Set the initial slot.
	k.SetCurrentSlot(ctx, gs.CurrentSlot)

	k.logger.Info("SVM genesis initialized",
		"accounts", len(gs.Accounts),
		"programs", len(gs.Programs),
		"slot", gs.CurrentSlot,
	)
}

// ExportGenesis exports the SVM module's current state as a genesis state.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:      k.GetParams(ctx),
		Accounts:    k.GetAllAccounts(ctx),
		Programs:    k.GetAllProgramMetas(ctx),
		CurrentSlot: k.GetCurrentSlot(ctx),
	}
}
