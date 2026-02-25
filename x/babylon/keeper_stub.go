//go:build !proprietary

package babylon

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/babylon/types"
)

// StubKeeper is a no-op implementation of BabylonKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub babylon keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger                                          { return k.logger }
func (k *StubKeeper) GetConfig(_ sdk.Context) types.BTCRestakingConfig            { return types.DefaultBTCRestakingConfig() }
func (k *StubKeeper) SetConfig(_ sdk.Context, _ types.BTCRestakingConfig) error   { return nil }
func (k *StubKeeper) IsEnabled(_ sdk.Context) bool                                { return false }
func (k *StubKeeper) GetStakingPosition(_ sdk.Context, _ string) (types.BTCStakingPosition, bool) {
	return types.BTCStakingPosition{}, false
}
func (k *StubKeeper) SetStakingPosition(_ sdk.Context, _ types.BTCStakingPosition) error { return nil }
func (k *StubKeeper) GetAllPositions(_ sdk.Context) []types.BTCStakingPosition           { return nil }
func (k *StubKeeper) GetCheckpoint(_ sdk.Context, _ uint64) (types.BTCCheckpoint, bool) {
	return types.BTCCheckpoint{}, false
}
func (k *StubKeeper) SetCheckpoint(_ sdk.Context, _ types.BTCCheckpoint) error { return nil }
func (k *StubKeeper) GetCurrentEpoch(_ sdk.Context) uint64                     { return 0 }
func (k *StubKeeper) GetEpochSnapshot(_ sdk.Context, _ uint64) (types.BabylonEpochSnapshot, bool) {
	return types.BabylonEpochSnapshot{}, false
}
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
