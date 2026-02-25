//go:build !proprietary

package burn

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/burn/types"
)

// StubKeeper is a no-op implementation of BurnKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub burn keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger                    { return k.logger }
func (k *StubKeeper) GetParams(_ sdk.Context) types.Params  { return types.DefaultParams() }
func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error { return nil }
func (k *StubKeeper) BurnFromSource(_ sdk.Context, _ types.BurnSource, _ math.Int, _ string) error {
	return nil
}
func (k *StubKeeper) GetTotalBurned(_ sdk.Context) math.Int              { return math.ZeroInt() }
func (k *StubKeeper) GetBurnStats(_ sdk.Context) types.BurnStats         { return types.DefaultBurnStats() }
func (k *StubKeeper) GetBurnRecords(_ sdk.Context, _ int) []types.BurnRecord { return nil }
func (k *StubKeeper) DistributeFees(_ sdk.Context) error                      { return nil }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState)         {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
