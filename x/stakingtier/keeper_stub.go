//go:build !full

package stakingtier

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/stakingtier/types"
)

// StubKeeper is a no-op implementation of StakingTierKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub staking tier keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{logger: logger.With("module", types.ModuleName)}
}

func (k *StubKeeper) Logger() log.Logger                              { return k.logger }
func (k *StubKeeper) GetParams(_ sdk.Context) types.Params            { return types.DefaultParams() }
func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error   { return nil }
func (k *StubKeeper) GetPool(_ sdk.Context) types.Pool                { return types.DefaultPool() }
func (k *StubKeeper) CheckStakingMsg(_ sdk.Context, _ sdk.Msg) error  { return nil }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
func (k *StubKeeper) SettleEpochBonus(_ sdk.Context, _, _ math.Int, _ bool, _ int64,
	_ func(sdk.Context, math.Int) error) (math.Int, error) {
	return math.ZeroInt(), nil
}
