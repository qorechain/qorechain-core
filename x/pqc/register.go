//go:build proprietary

package pqc

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/ffi"
	"github.com/qorechain/qorechain-core/x/pqc/keeper"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the PQCKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) PQCClient() PQCClient                                        { return a.k.PQCClient() }
func (a *keeperAdapter) Logger() log.Logger                                           { return a.k.Logger() }
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params                       { return a.k.GetParams(ctx) }
func (a *keeperAdapter) SetParams(ctx sdk.Context, p types.Params) error              { return a.k.SetParams(ctx, p) }
func (a *keeperAdapter) GetPQCAccount(ctx sdk.Context, addr string) (types.PQCAccountInfo, bool) {
	return a.k.GetPQCAccount(ctx, addr)
}
func (a *keeperAdapter) HasPQCAccount(ctx sdk.Context, addr string) bool              { return a.k.HasPQCAccount(ctx, addr) }
func (a *keeperAdapter) SetPQCAccount(ctx sdk.Context, info types.PQCAccountInfo) error {
	return a.k.SetPQCAccount(ctx, info)
}
func (a *keeperAdapter) IncrementPQCVerifications(ctx sdk.Context)                    { a.k.IncrementPQCVerifications(ctx) }
func (a *keeperAdapter) IncrementClassicalFallbacks(ctx sdk.Context)                  { a.k.IncrementClassicalFallbacks(ctx) }
func (a *keeperAdapter) GetStats(ctx sdk.Context) types.PQCStats                      { return a.k.GetStats(ctx) }
func (a *keeperAdapter) SetStats(ctx sdk.Context, s types.PQCStats)                   { a.k.SetStats(ctx, s) }
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState)            { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState             { return a.k.ExportGenesis(ctx) }

// RealNewPQCClient creates the real FFI client.
func RealNewPQCClient() PQCClient {
	return ffi.NewFFIClient()
}

// RealNewPQCKeeper creates the real PQC keeper wrapped in the adapter.
func RealNewPQCKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, client PQCClient, logger log.Logger) PQCKeeper {
	ffiClient, ok := client.(ffi.PQCClient)
	if !ok {
		panic("PQCClient must implement ffi.PQCClient in proprietary build")
	}
	k := keeper.NewKeeper(cdc, storeKey, ffiClient, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real PQC AppModule.
func RealNewAppModule(k PQCKeeper) AppModule {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("PQCKeeper must be a keeperAdapter in proprietary build")
	}
	return NewAppModule(adapter.k)
}

// RealNewPQCVerifyDecorator creates the real PQC ante decorator.
func RealNewPQCVerifyDecorator(k PQCKeeper, client PQCClient) PQCVerifyDecorator {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("PQCKeeper must be a keeperAdapter in proprietary build")
	}
	return NewPQCVerifyDecorator(adapter.k, client.(ffi.PQCClient))
}

// ExtractConcreteKeeper returns the underlying concrete keeper.Keeper from
// a PQCKeeper interface. Used by x/bridge to get the concrete type it needs.
func ExtractConcreteKeeper(k PQCKeeper) keeper.Keeper {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("PQCKeeper must be a keeperAdapter in proprietary build")
	}
	return adapter.k
}
