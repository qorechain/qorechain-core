//go:build !proprietary

package rdk

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// StubKeeper is a no-op implementation of RDKKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub rdk keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger { return k.logger }

// Rollup Lifecycle
func (k *StubKeeper) CreateRollup(_ sdk.Context, _ types.RollupConfig) (*types.RollupConfig, error) {
	return nil, types.ErrRollupNotActive
}
func (k *StubKeeper) PauseRollup(_ sdk.Context, _ string, _ string) error  { return nil }
func (k *StubKeeper) ResumeRollup(_ sdk.Context, _ string) error           { return nil }
func (k *StubKeeper) StopRollup(_ sdk.Context, _ string) error             { return nil }
func (k *StubKeeper) GetRollup(_ sdk.Context, _ string) (*types.RollupConfig, error) {
	return nil, types.ErrRollupNotFound
}
func (k *StubKeeper) ListRollups(_ sdk.Context) ([]*types.RollupConfig, error) {
	return nil, nil
}
func (k *StubKeeper) ListRollupsByCreator(_ sdk.Context, _ string) ([]*types.RollupConfig, error) {
	return nil, nil
}

// Settlement
func (k *StubKeeper) SubmitBatch(_ sdk.Context, _ types.SettlementBatch) error { return nil }
func (k *StubKeeper) ChallengeBatch(_ sdk.Context, _ string, _ uint64, _ []byte) error {
	return nil
}
func (k *StubKeeper) FinalizeBatch(_ sdk.Context, _ string, _ uint64) error { return nil }
func (k *StubKeeper) GetBatch(_ sdk.Context, _ string, _ uint64) (*types.SettlementBatch, error) {
	return nil, types.ErrBatchNotFound
}
func (k *StubKeeper) GetLatestBatch(_ sdk.Context, _ string) (*types.SettlementBatch, error) {
	return nil, types.ErrBatchNotFound
}

// DA Routing
func (k *StubKeeper) SubmitDABlob(_ sdk.Context, _ types.DABlob) (*types.DACommitment, error) {
	return nil, nil
}
func (k *StubKeeper) GetDABlob(_ sdk.Context, _ string, _ uint64) (*types.DABlob, error) {
	return nil, types.ErrDABlobNotFound
}
func (k *StubKeeper) PruneExpiredBlobs(_ sdk.Context) (uint64, error) { return 0, nil }

// AI-Assisted Configuration
func (k *StubKeeper) SuggestProfile(_ sdk.Context, _ string) (*types.RollupProfile, error) {
	p := types.ProfileDeFi
	return &p, nil
}
func (k *StubKeeper) OptimizeGasConfig(_ sdk.Context, _ string) (*types.RollupGasConfig, error) {
	gc := types.DefaultRollupGasConfig()
	return &gc, nil
}

// Params / Genesis
func (k *StubKeeper) GetParams(_ sdk.Context) types.Params       { return types.DefaultParams() }
func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error { return nil }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
