//go:build !proprietary

package rlconsensus

import (
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// StubKeeper is a no-op implementation of RLConsensusKeeper for the public build.
// RL consensus functionality is only available in the proprietary build.
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

func (k *StubKeeper) GetAgentStatus(_ sdk.Context) types.AgentStatus {
	return types.AgentStatus{Mode: types.AgentModePaused}
}

func (k *StubKeeper) GetLatestObservation(_ sdk.Context) (*types.Observation, error) {
	return nil, types.ErrRLDisabled
}

func (k *StubKeeper) GetLatestReward(_ sdk.Context) (*types.Reward, error) {
	return nil, types.ErrRLDisabled
}

func (k *StubKeeper) GetPolicyWeights(_ sdk.Context) (*types.PolicyWeights, error) {
	return nil, types.ErrRLDisabled
}

func (k *StubKeeper) GetCurrentBlockTime(_ sdk.Context) time.Duration {
	return 5 * time.Second
}

func (k *StubKeeper) GetCurrentBaseGasPrice(_ sdk.Context) math.LegacyDec {
	return math.LegacyNewDec(100)
}

func (k *StubKeeper) GetValidatorSetSize(_ sdk.Context) uint64 {
	return 100
}

func (k *StubKeeper) GetCurrentEpoch(_ sdk.Context) uint64 {
	return 0
}

func (k *StubKeeper) IsRLActive(_ sdk.Context) bool {
	return false
}

func (k *StubKeeper) BeginBlock(_ sdk.Context) error {
	return nil
}

func (k *StubKeeper) EndBlock(_ sdk.Context) error {
	return nil
}

func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}

func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesis()
}

func (k *StubKeeper) Logger() log.Logger {
	return k.logger
}
