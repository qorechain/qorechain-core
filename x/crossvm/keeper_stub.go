//go:build !proprietary

package crossvm

import (
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// StubKeeper is a no-op implementation of CrossVMKeeper for the public build.
// Cross-VM functionality is only available in the proprietary build.
type StubKeeper struct {
	logger log.Logger
}

func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{logger: logger}
}

func (k *StubKeeper) GetParams(_ sdk.Context) types.Params {
	return types.DefaultParams()
}

func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error {
	return nil
}

func (k *StubKeeper) SubmitMessage(_ sdk.Context, _ types.CrossVMMessage) (string, error) {
	return "", types.ErrUnsupportedVM.Wrap("cross-VM not available in public build")
}

func (k *StubKeeper) GetMessage(_ sdk.Context, _ string) (types.CrossVMMessage, bool) {
	return types.CrossVMMessage{}, false
}

func (k *StubKeeper) GetPendingMessages(_ sdk.Context) []types.CrossVMMessage {
	return nil
}

func (k *StubKeeper) ProcessQueue(_ sdk.Context) error {
	return nil
}

func (k *StubKeeper) ExecuteSyncCall(_ sdk.Context, _ types.CrossVMMessage) (types.CrossVMResponse, error) {
	return types.CrossVMResponse{}, types.ErrUnsupportedVM.Wrap("cross-VM not available in public build")
}

func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}

func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
