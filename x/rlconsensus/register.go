//go:build proprietary

package rlconsensus

import (
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/rlconsensus/keeper"
	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the RLConsensusKeeper interface.
type keeperAdapter struct {
	k *keeper.Keeper
}

// --- Params ---

func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params {
	return a.k.GetParams(ctx)
}

func (a *keeperAdapter) SetParams(ctx sdk.Context, params types.Params) error {
	return a.k.SetParams(ctx, params)
}

// --- Agent Status ---

func (a *keeperAdapter) GetAgentStatus(ctx sdk.Context) types.AgentStatus {
	return a.k.GetAgentStatus(ctx)
}

// --- Observations ---

func (a *keeperAdapter) GetLatestObservation(ctx sdk.Context) (*types.Observation, error) {
	return a.k.GetLatestObservation(ctx)
}

// --- Rewards ---

func (a *keeperAdapter) GetLatestReward(ctx sdk.Context) (*types.Reward, error) {
	return a.k.GetLatestReward(ctx)
}

// --- Policy ---

func (a *keeperAdapter) GetPolicyWeights(ctx sdk.Context) (*types.PolicyWeights, error) {
	return a.k.GetPolicyWeights(ctx)
}

// --- RLConsensusParamsProvider interface ---

func (a *keeperAdapter) GetCurrentBlockTime(ctx sdk.Context) time.Duration {
	applied := a.k.GetAppliedParams(ctx)
	if applied.BlockTimeMs > 0 {
		return time.Duration(applied.BlockTimeMs) * time.Millisecond
	}
	params := a.k.GetParams(ctx)
	return time.Duration(params.DefaultBlockTimeMs) * time.Millisecond
}

func (a *keeperAdapter) GetCurrentBaseGasPrice(ctx sdk.Context) math.LegacyDec {
	applied := a.k.GetAppliedParams(ctx)
	d, err := math.LegacyNewDecFromStr(applied.GasPriceFloor)
	if err != nil {
		return math.LegacyNewDec(100) // fallback default
	}
	return d
}

func (a *keeperAdapter) GetValidatorSetSize(ctx sdk.Context) uint64 {
	params := a.k.GetParams(ctx)
	return params.DefaultValidatorSetSize
}

func (a *keeperAdapter) GetCurrentEpoch(ctx sdk.Context) uint64 {
	status := a.k.GetAgentStatus(ctx)
	return status.CurrentEpoch
}

func (a *keeperAdapter) IsRLActive(ctx sdk.Context) bool {
	params := a.k.GetParams(ctx)
	return params.Enabled && params.AgentMode != types.AgentModePaused
}

// --- ABCI hooks ---

func (a *keeperAdapter) BeginBlock(ctx sdk.Context) error {
	return a.k.BeginBlock(ctx)
}

func (a *keeperAdapter) EndBlock(ctx sdk.Context) error {
	return a.k.EndBlock(ctx)
}

// --- Genesis ---

func (a *keeperAdapter) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	a.k.InitGenesis(ctx, gs)
}

func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return a.k.ExportGenesis(ctx)
}

// --- Advisory rollup configuration ---

func (a *keeperAdapter) SuggestRollupProfile(_ sdk.Context, _ string) (string, error) {
	return "defi", nil
}

func (a *keeperAdapter) OptimizeRollupGas(_ sdk.Context, _ map[string]uint64) (uint64, error) {
	return 0, nil
}

// --- Logger ---

func (a *keeperAdapter) Logger() log.Logger {
	return a.k.Logger()
}

// --- Message handlers (exposed for CLI/gRPC wiring) ---

// HandleMsgSetAgentMode delegates to the concrete keeper.
func (a *keeperAdapter) HandleMsgSetAgentMode(ctx sdk.Context, msg *types.MsgSetAgentMode) error {
	return a.k.HandleMsgSetAgentMode(ctx, msg)
}

// HandleMsgResumeAgent delegates to the concrete keeper.
func (a *keeperAdapter) HandleMsgResumeAgent(ctx sdk.Context, msg *types.MsgResumeAgent) error {
	return a.k.HandleMsgResumeAgent(ctx, msg)
}

// HandleMsgUpdatePolicy delegates to the concrete keeper.
func (a *keeperAdapter) HandleMsgUpdatePolicy(ctx sdk.Context, msg *types.MsgUpdatePolicy) error {
	return a.k.HandleMsgUpdatePolicy(ctx, msg)
}

// HandleMsgUpdateRewardWeights delegates to the concrete keeper.
func (a *keeperAdapter) HandleMsgUpdateRewardWeights(ctx sdk.Context, msg *types.MsgUpdateRewardWeights) error {
	return a.k.HandleMsgUpdateRewardWeights(ctx, msg)
}

// --- Factory functions ---

// RealNewRLConsensusKeeper creates the proprietary RL consensus keeper.
func RealNewRLConsensusKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
) RLConsensusKeeper {
	k := keeper.NewKeeper(cdc, storeKey, logger)
	return &keeperAdapter{k: k}
}

// RealNewRLConsensusKeeperWithDeps creates the keeper and returns the underlying
// keeper pointer for wiring cross-module dependencies via setter methods.
func RealNewRLConsensusKeeperWithDeps(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
) (RLConsensusKeeper, *keeper.Keeper) {
	k := keeper.NewKeeper(cdc, storeKey, logger)
	return &keeperAdapter{k: k}, k
}

// RealNewAppModule creates the proprietary AppModule backed by the real keeper.
func RealNewAppModule(k RLConsensusKeeper) module.AppModule {
	return NewProprietaryAppModule(k)
}
