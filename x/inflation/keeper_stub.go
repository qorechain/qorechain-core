//go:build !proprietary

package inflation

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/inflation/types"
)

// StubKeeper is a no-op implementation of InflationKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub inflation keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger                                   { return k.logger }
func (k *StubKeeper) GetParams(_ sdk.Context) types.Params                 { return types.DefaultParams() }
func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error        { return nil }
func (k *StubKeeper) GetCurrentInflationRate(_ sdk.Context) math.LegacyDec { return math.LegacyZeroDec() }
func (k *StubKeeper) GetEpochInfo(_ sdk.Context) types.EpochInfo           { return types.DefaultEpochInfo() }
func (k *StubKeeper) MintEpochEmission(_ sdk.Context) error                { return nil }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState)      {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
