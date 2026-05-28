//go:build !full

package lightnode

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/lightnode/types"
)

// StubKeeper is a no-op implementation of LightNodeKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub light node keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger                                           { return k.logger }
func (k *StubKeeper) GetParams(_ sdk.Context) types.Params                         { return types.DefaultParams() }
func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error                { return nil }
func (k *StubKeeper) RegisterLightNode(_ sdk.Context, _ types.LightNodeInfo) error { return nil }
func (k *StubKeeper) DeregisterLightNode(_ sdk.Context, _ string) error            { return nil }
func (k *StubKeeper) GetLightNode(_ sdk.Context, _ string) (types.LightNodeInfo, bool) {
	return types.LightNodeInfo{}, false
}
func (k *StubKeeper) GetAllLightNodes(_ sdk.Context) []types.LightNodeInfo                { return nil }
func (k *StubKeeper) GetLightNodesByStatus(_ sdk.Context, _ string) []types.LightNodeInfo { return nil }
func (k *StubKeeper) GetLightNodeCount(_ sdk.Context) uint64                              { return 0 }
func (k *StubKeeper) RecordHeartbeat(_ sdk.Context, _ string, _ int64) error              { return nil }
func (k *StubKeeper) DistributeRewards(_ sdk.Context, _ math.Int) error                   { return nil }
func (k *StubKeeper) GetAccumulatedRewards(_ sdk.Context, _ string) math.Int              { return math.ZeroInt() }
func (k *StubKeeper) ClaimRewards(_ sdk.Context, _ string) (math.Int, error) {
	return math.ZeroInt(), nil
}
func (k *StubKeeper) GetStats(_ sdk.Context) types.LightNodeStats {
	return types.DefaultLightNodeStats()
}
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
