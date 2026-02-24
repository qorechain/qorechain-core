//go:build !proprietary

package svm

import (
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// StubKeeper is a no-op implementation of SVMKeeper for the public build.
// SVM functionality is only available in the proprietary build.
type StubKeeper struct {
	logger log.Logger
}

func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{logger: logger}
}

func (k *StubKeeper) GetAccount(_ sdk.Context, _ [32]byte) (*types.SVMAccount, error) {
	return nil, types.ErrSVMDisabled
}

func (k *StubKeeper) SetAccount(_ sdk.Context, _ *types.SVMAccount) error {
	return types.ErrSVMDisabled
}

func (k *StubKeeper) DeleteAccount(_ sdk.Context, _ [32]byte) error {
	return types.ErrSVMDisabled
}

func (k *StubKeeper) DeployProgram(_ sdk.Context, _ [32]byte, _ []byte) ([32]byte, error) {
	return [32]byte{}, types.ErrSVMDisabled
}

func (k *StubKeeper) ExecuteProgram(_ sdk.Context, _ [32]byte, _ []byte,
	_ []types.AccountMeta, _ [][32]byte) (*types.ExecutionResult, error) {
	return nil, types.ErrSVMDisabled
}

func (k *StubKeeper) SVMToCosmosAddr(svmAddr [32]byte) sdk.AccAddress {
	return types.SVMToCosmosAddress(svmAddr)
}

func (k *StubKeeper) CosmosToSVMAddr(_ sdk.AccAddress) ([32]byte, error) {
	return [32]byte{}, types.ErrSVMDisabled
}

func (k *StubKeeper) CollectRent(_ sdk.Context, _ [32]byte) error {
	return types.ErrSVMDisabled
}

func (k *StubKeeper) GetMinimumBalance(dataLen uint64) uint64 {
	// Return a reasonable default for queries even in public build.
	// NOTE: truncates float to uint; acceptable for stub approximation.
	return (128 + dataLen) * types.DefaultLamportsPerByte * uint64(types.DefaultRentExemptionMulti)
}

func (k *StubKeeper) GetCurrentSlot(_ sdk.Context) uint64 {
	return 0
}

func (k *StubKeeper) GetParams(_ sdk.Context) types.Params {
	return types.DefaultParams()
}

func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error {
	return nil
}

func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}

func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesis()
}

func (k *StubKeeper) Logger() log.Logger {
	return k.logger
}
