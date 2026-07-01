//go:build !full

package svm

import (
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// StubKeeper is a no-op implementation of SVMKeeper for the public build.
// SVM functionality is only available in the full build.
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

func (k *StubKeeper) SetBankKeeper(_ types.BankKeeper) {}

func (k *StubKeeper) GetNativeLamports(_ sdk.Context, _ [32]byte) uint64 {
	return 0
}

func (k *StubKeeper) FaucetCreditNative(_ sdk.Context, _ [32]byte, _ uint64) error {
	return types.ErrSVMDisabled
}

func (k *StubKeeper) SetAuthenticatorResolver(_ types.AuthenticatorResolver) {}

func (k *StubKeeper) ResolveAuthenticatedSigner(_ sdk.Context, _ string, _, _, _ []byte) ([32]byte, bool) {
	return [32]byte{}, false
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
	base := (128 + dataLen) * types.DefaultLamportsPerByte
	intBase := sdkmath.NewIntFromUint64(base)
	return types.DefaultRentExemptionMultiDec.MulInt(intBase).TruncateInt().Uint64()
}

func (k *StubKeeper) GetCurrentSlot(_ sdk.Context) uint64 {
	return 0
}
func (k *StubKeeper) BeginBlock(_ sdk.Context)                          {}
func (k *StubKeeper) IsRecentBlockhash(_ sdk.Context, _ []byte) bool    { return false }
func (k *StubKeeper) GetLatestBlockhash(_ sdk.Context) ([]byte, uint64) { return nil, 0 }
func (k *StubKeeper) GetSignaturesForAddress(_ sdk.Context, _ [32]byte, _ int) []string {
	return nil
}
func (k *StubKeeper) GetSVMTransaction(_ sdk.Context, _ string) (*types.SVMTxRecord, bool) {
	return nil, false
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

func (k *StubKeeper) IterateAccounts(_ sdk.Context, _ func(types.SVMAccount) bool) {}

func (k *StubKeeper) GetAllAccounts(_ sdk.Context) []types.SVMAccount {
	return nil
}

func (k *StubKeeper) Logger() log.Logger {
	return k.logger
}
