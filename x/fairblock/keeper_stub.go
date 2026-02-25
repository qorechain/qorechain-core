//go:build !proprietary

package fairblock

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/fairblock/types"
)

// StubKeeper is a no-op implementation of FairBlockKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub fairblock keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger                                    { return k.logger }
func (k *StubKeeper) GetConfig(_ sdk.Context) types.FairBlockConfig         { return types.DefaultFairBlockConfig() }
func (k *StubKeeper) SetConfig(_ sdk.Context, _ types.FairBlockConfig) error { return nil }
func (k *StubKeeper) IsEnabled(_ sdk.Context) bool                          { return false }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState)       {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
