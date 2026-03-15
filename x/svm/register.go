//go:build proprietary

package svm

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	"github.com/qorechain/qorechain-core/x/svm/ffi"
	"github.com/qorechain/qorechain-core/x/svm/keeper"
	"github.com/qorechain/qorechain-core/x/svm/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the SVMKeeper interface.
type keeperAdapter struct {
	k *keeper.Keeper
}

func (a *keeperAdapter) GetAccount(ctx sdk.Context, addr [32]byte) (*types.SVMAccount, error) {
	return a.k.GetAccount(ctx, addr)
}
func (a *keeperAdapter) SetAccount(ctx sdk.Context, account *types.SVMAccount) error {
	return a.k.SetAccount(ctx, account)
}
func (a *keeperAdapter) DeleteAccount(ctx sdk.Context, addr [32]byte) error {
	return a.k.DeleteAccount(ctx, addr)
}
func (a *keeperAdapter) DeployProgram(ctx sdk.Context, deployer [32]byte, bytecode []byte) ([32]byte, error) {
	return a.k.DeployProgram(ctx, deployer, bytecode)
}
func (a *keeperAdapter) ExecuteProgram(ctx sdk.Context, programID [32]byte, instruction []byte,
	accounts []types.AccountMeta, signers [][32]byte) (*types.ExecutionResult, error) {
	return a.k.ExecuteProgram(ctx, programID, instruction, accounts, signers)
}
func (a *keeperAdapter) SVMToCosmosAddr(svmAddr [32]byte) sdk.AccAddress {
	return a.k.SVMToCosmosAddr(svmAddr)
}
func (a *keeperAdapter) CosmosToSVMAddr(cosmosAddr sdk.AccAddress) ([32]byte, error) {
	return a.k.CosmosToSVMAddr(cosmosAddr)
}
func (a *keeperAdapter) CollectRent(ctx sdk.Context, addr [32]byte) error {
	return a.k.CollectRent(ctx, addr)
}
func (a *keeperAdapter) GetMinimumBalance(dataLen uint64) uint64 {
	return a.k.GetMinimumBalance(dataLen)
}
func (a *keeperAdapter) GetCurrentSlot(ctx sdk.Context) uint64 {
	return a.k.GetCurrentSlot(ctx)
}
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params {
	return a.k.GetParams(ctx)
}
func (a *keeperAdapter) SetParams(ctx sdk.Context, params types.Params) error {
	return a.k.SetParams(ctx, params)
}
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	a.k.InitGenesis(ctx, gs)
}
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return a.k.ExportGenesis(ctx)
}
func (a *keeperAdapter) IterateAccounts(ctx sdk.Context, cb func(types.SVMAccount) bool) {
	a.k.IterateAccounts(ctx, cb)
}
func (a *keeperAdapter) GetAllAccounts(ctx sdk.Context) []types.SVMAccount {
	return a.k.GetAllAccounts(ctx)
}
func (a *keeperAdapter) Logger() log.Logger {
	return a.k.Logger()
}

// RealNewSVMKeeper creates the real SVM keeper backed by the BPF executor.
func RealNewSVMKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	pqcKeeper pqcmod.PQCKeeper,
	aiKeeper aimod.AIKeeper,
	crossvmKeeper crossvmmod.CrossVMKeeper,
	logger log.Logger,
) SVMKeeper {
	k := keeper.NewKeeper(cdc, storeKey, pqcKeeper, aiKeeper, crossvmKeeper, logger)

	// Initialize the BPF execution engine via the Rust FFI bridge.
	exec := ffi.NewFFIExecutor(types.DefaultComputeBudgetMax)
	k.SetExecutor(exec)

	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real SVM AppModule.
func RealNewAppModule(k SVMKeeper) module.AppModule {
	return NewProprietaryAppModule(k)
}
