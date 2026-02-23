//go:build proprietary

package crossvm

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	evmkeeper "github.com/cosmos/evm/x/vm/keeper"

	"github.com/qorechain/qorechain-core/x/crossvm/keeper"
	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the CrossVMKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params                                  { return a.k.GetParams(ctx) }
func (a *keeperAdapter) SetParams(ctx sdk.Context, p types.Params) error                         { return a.k.SetParams(ctx, p) }
func (a *keeperAdapter) SubmitMessage(ctx sdk.Context, msg types.CrossVMMessage) (string, error)  { return a.k.SubmitMessage(ctx, msg) }
func (a *keeperAdapter) GetMessage(ctx sdk.Context, id string) (types.CrossVMMessage, bool)       { return a.k.GetMessage(ctx, id) }
func (a *keeperAdapter) GetPendingMessages(ctx sdk.Context) []types.CrossVMMessage                { return a.k.GetPendingMessages(ctx) }
func (a *keeperAdapter) ProcessQueue(ctx sdk.Context) error                                       { return a.k.ProcessQueue(ctx) }
func (a *keeperAdapter) ExecuteSyncCall(ctx sdk.Context, msg types.CrossVMMessage) (types.CrossVMResponse, error) {
	return a.k.ExecuteSyncCall(ctx, msg)
}
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState)   { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState    { return a.k.ExportGenesis(ctx) }

// RealNewCrossVMKeeper creates the real cross-VM keeper.
func RealNewCrossVMKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	evmKeeper *evmkeeper.Keeper,
	wasmKeeper *wasmkeeper.Keeper,
	logger log.Logger,
) CrossVMKeeper {
	k := keeper.NewKeeper(cdc, storeKey, evmKeeper, wasmKeeper, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real cross-VM AppModule.
func RealNewAppModule(k CrossVMKeeper) module.AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("CrossVMKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}
