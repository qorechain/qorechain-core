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

func (a *keeperAdapter) PQCClient() PQCClient                                   { return a.k.PQCClient() }
func (a *keeperAdapter) Logger() log.Logger                                      { return a.k.Logger() }
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params                  { return a.k.GetParams(ctx) }
func (a *keeperAdapter) SetParams(ctx sdk.Context, p types.Params) error         { return a.k.SetParams(ctx, p) }
func (a *keeperAdapter) GetPQCAccount(ctx sdk.Context, addr string) (types.PQCAccountInfo, bool) {
	return a.k.GetPQCAccount(ctx, addr)
}
func (a *keeperAdapter) HasPQCAccount(ctx sdk.Context, addr string) bool { return a.k.HasPQCAccount(ctx, addr) }
func (a *keeperAdapter) SetPQCAccount(ctx sdk.Context, info types.PQCAccountInfo) error {
	return a.k.SetPQCAccount(ctx, info)
}
func (a *keeperAdapter) IncrementPQCVerifications(ctx sdk.Context)  { a.k.IncrementPQCVerifications(ctx) }
func (a *keeperAdapter) IncrementClassicalFallbacks(ctx sdk.Context) { a.k.IncrementClassicalFallbacks(ctx) }
func (a *keeperAdapter) IncrementMLKEMOperations(ctx sdk.Context)   { a.k.IncrementMLKEMOperations(ctx) }
func (a *keeperAdapter) GetStats(ctx sdk.Context) types.PQCStats    { return a.k.GetStats(ctx) }
func (a *keeperAdapter) SetStats(ctx sdk.Context, s types.PQCStats) { a.k.SetStats(ctx, s) }

// Hybrid signature methods (v1.1.0)
func (a *keeperAdapter) GetHybridSignatureMode(ctx sdk.Context) types.HybridSignatureMode {
	return a.k.GetHybridSignatureMode(ctx)
}
func (a *keeperAdapter) IncrementHybridVerifications(ctx sdk.Context) {
	a.k.IncrementHybridVerifications(ctx)
}

// Algorithm registry (v0.6.0)
func (a *keeperAdapter) RegisterAlgorithm(ctx sdk.Context, algo types.AlgorithmInfo) error {
	return a.k.RegisterAlgorithm(ctx, algo)
}
func (a *keeperAdapter) GetAlgorithm(ctx sdk.Context, id types.AlgorithmID) (types.AlgorithmInfo, error) {
	return a.k.GetAlgorithm(ctx, id)
}
func (a *keeperAdapter) ListAlgorithms(ctx sdk.Context) []types.AlgorithmInfo {
	return a.k.ListAlgorithms(ctx)
}
func (a *keeperAdapter) UpdateAlgorithmStatus(ctx sdk.Context, id types.AlgorithmID, status types.AlgorithmStatus) error {
	return a.k.UpdateAlgorithmStatus(ctx, id, status)
}
func (a *keeperAdapter) GetActiveSignatureAlgorithms(ctx sdk.Context) []types.AlgorithmInfo {
	return a.k.GetActiveSignatureAlgorithms(ctx)
}
func (a *keeperAdapter) GetActiveKEMAlgorithms(ctx sdk.Context) []types.AlgorithmInfo {
	return a.k.GetActiveKEMAlgorithms(ctx)
}

// Migration (v0.6.0)
func (a *keeperAdapter) GetMigration(ctx sdk.Context, fromID types.AlgorithmID) (types.MigrationInfo, bool) {
	return a.k.GetMigration(ctx, fromID)
}
func (a *keeperAdapter) SetMigration(ctx sdk.Context, migration types.MigrationInfo) error {
	return a.k.SetMigration(ctx, migration)
}
func (a *keeperAdapter) DeleteMigration(ctx sdk.Context, fromID types.AlgorithmID) {
	a.k.DeleteMigration(ctx, fromID)
}

// Genesis
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) { a.k.InitGenesis(ctx, gs) }
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState  { return a.k.ExportGenesis(ctx) }

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

// RealNewPQCHybridVerifyDecorator creates the real hybrid PQC ante decorator.
func RealNewPQCHybridVerifyDecorator(k PQCKeeper, client PQCClient) PQCHybridVerifyDecorator {
	adapter, ok := k.(*keeperAdapter)
	if !ok {
		panic("PQCKeeper must be a keeperAdapter in proprietary build")
	}
	return NewPQCHybridVerifyDecorator(adapter.k, client.(ffi.PQCClient))
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
