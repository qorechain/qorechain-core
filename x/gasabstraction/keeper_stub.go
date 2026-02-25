//go:build !proprietary

package gasabstraction

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// StubKeeper is a no-op implementation of GasAbstractionKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub gas abstraction keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger                                             { return k.logger }
func (k *StubKeeper) GetConfig(_ sdk.Context) types.GasAbstractionConfig             { return types.DefaultGasAbstractionConfig() }
func (k *StubKeeper) SetConfig(_ sdk.Context, _ types.GasAbstractionConfig) error    { return nil }
func (k *StubKeeper) IsEnabled(_ sdk.Context) bool                                   { return true }
func (k *StubKeeper) GetAcceptedTokens(_ sdk.Context) []types.AcceptedFeeToken       { return types.DefaultGasAbstractionConfig().AcceptedTokens }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState)                {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
