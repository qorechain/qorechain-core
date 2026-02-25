//go:build !proprietary

package abstractaccount

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// StubKeeper is a no-op implementation of AbstractAccountKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub abstract account keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger                                                  { return k.logger }
func (k *StubKeeper) GetConfig(_ sdk.Context) types.AbstractAccountConfig                 { return types.DefaultAbstractAccountConfig() }
func (k *StubKeeper) SetConfig(_ sdk.Context, _ types.AbstractAccountConfig) error         { return nil }
func (k *StubKeeper) IsEnabled(_ sdk.Context) bool                                        { return false }
func (k *StubKeeper) GetAccount(_ sdk.Context, _ string) (types.AbstractAccount, bool)    { return types.AbstractAccount{}, false }
func (k *StubKeeper) SetAccount(_ sdk.Context, _ types.AbstractAccount) error              { return nil }
func (k *StubKeeper) GetAllAccounts(_ sdk.Context) []types.AbstractAccount                 { return nil }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState)                      {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
