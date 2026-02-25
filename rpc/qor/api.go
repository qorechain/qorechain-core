//go:build proprietary

// Package qor implements the qor_ JSON-RPC namespace for QoreChain-specific queries.
// This provides endpoints for PQC key status, AI module stats, cross-VM message status,
// validator reputation, and multilayer chain info.
package qor

import (
	"context"
	"fmt"

	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
)

// QorAPI defines the qor_ JSON-RPC namespace methods.
type QorAPI struct {
	ctx              context.Context
	logger           log.Logger
	pqcKeeper        pqcmod.PQCKeeper
	aiKeeper         aimod.AIKeeper
	crossvmKeeper    crossvmmod.CrossVMKeeper
	reputationKeeper reputationkeeper.Keeper
	bridgeKeeper        bridgemod.BridgeKeeper
	multilayerKeeper    multilayermod.MultilayerKeeper
	rlconsensusKeeper   rlconsensusmod.RLConsensusKeeper
}

// NewQorAPI creates a new QorAPI instance.
func NewQorAPI(
	ctx context.Context,
	logger log.Logger,
	pqcKeeper pqcmod.PQCKeeper,
	aiKeeper aimod.AIKeeper,
	crossvmKeeper crossvmmod.CrossVMKeeper,
	reputationKeeper reputationkeeper.Keeper,
	bridgeKeeper bridgemod.BridgeKeeper,
	multilayerKeeper multilayermod.MultilayerKeeper,
	rlconsensusKeeper rlconsensusmod.RLConsensusKeeper,
) *QorAPI {
	return &QorAPI{
		ctx:               ctx,
		logger:            logger.With("module", "qor-rpc"),
		pqcKeeper:         pqcKeeper,
		aiKeeper:          aiKeeper,
		crossvmKeeper:     crossvmKeeper,
		reputationKeeper:  reputationKeeper,
		bridgeKeeper:      bridgeKeeper,
		multilayerKeeper:  multilayerKeeper,
		rlconsensusKeeper: rlconsensusKeeper,
	}
}

// PQCKeyStatusResult contains the PQC key registration status for an address.
type PQCKeyStatusResult struct {
	Address         string `json:"address"`
	HasPQCKey       bool   `json:"has_pqc_key"`
	KeyType         string `json:"key_type,omitempty"`
	CreatedAtHeight int64  `json:"created_at_height,omitempty"`
}

// GetPQCKeyStatus returns the PQC key registration status for a given address.
func (api *QorAPI) GetPQCKeyStatus(address string) (*PQCKeyStatusResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	result := &PQCKeyStatusResult{Address: address}

	hasAccount := api.pqcKeeper.HasPQCAccount(sdkCtx, address)
	result.HasPQCKey = hasAccount

	if hasAccount {
		if acct, found := api.pqcKeeper.GetPQCAccount(sdkCtx, address); found {
			result.KeyType = acct.KeyType
			result.CreatedAtHeight = acct.CreatedAtHeight
		}
	}

	return result, nil
}

// AIStatsResult contains AI module statistics.
type AIStatsResult struct {
	TxsRouted          uint64  `json:"txs_routed"`
	AnomaliesDetected  uint64  `json:"anomalies_detected"`
	TxsFlagged         uint64  `json:"txs_flagged"`
	AnomalyThreshold   float64 `json:"anomaly_threshold"`
	RoutingStrategy    string  `json:"routing_strategy"`
}

// GetAIStats returns the AI module's current statistics and configuration.
func (api *QorAPI) GetAIStats() (*AIStatsResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	stats := api.aiKeeper.GetStats(sdkCtx)
	config := api.aiKeeper.GetConfig(sdkCtx)

	return &AIStatsResult{
		TxsRouted:         stats.TxsRouted,
		AnomaliesDetected: stats.AnomaliesDetected,
		TxsFlagged:        stats.TxsFlagged,
		AnomalyThreshold:  config.AnomalyThreshold,
		RoutingStrategy:   config.RoutingStrategy,
	}, nil
}

// CrossVMMessageResult contains cross-VM message status information.
type CrossVMMessageResult struct {
	MessageID      string `json:"message_id"`
	SourceVM       string `json:"source_vm"`
	TargetVM       string `json:"target_vm"`
	TargetContract string `json:"target_contract"`
	Status         string `json:"status"`
	CreatedHeight  int64  `json:"created_height"`
	ExecutedHeight int64  `json:"executed_height,omitempty"`
	Error          string `json:"error,omitempty"`
}

// GetCrossVMMessage returns the status of a cross-VM message by ID.
func (api *QorAPI) GetCrossVMMessage(msgID string) (*CrossVMMessageResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	msg, found := api.crossvmKeeper.GetMessage(sdkCtx, msgID)
	if !found {
		return nil, fmt.Errorf("cross-VM message not found: %s", msgID)
	}

	return &CrossVMMessageResult{
		MessageID:      msg.ID,
		SourceVM:       string(msg.SourceVM),
		TargetVM:       string(msg.TargetVM),
		TargetContract: msg.TargetContract,
		Status:         string(msg.Status),
		CreatedHeight:  msg.CreatedHeight,
		ExecutedHeight: msg.ExecutedHeight,
		Error:          msg.Error,
	}, nil
}

// ReputationScoreResult contains the reputation score for a validator.
type ReputationScoreResult struct {
	Validator         string  `json:"validator"`
	ReputationScore   float64 `json:"reputation_score"`
	StakeComponent    float64 `json:"stake_component"`
	PerfComponent     float64 `json:"perf_component"`
	ContribComponent  float64 `json:"contrib_component"`
	TimeComponent     float64 `json:"time_component"`
}

// GetReputationScore returns the reputation score for a validator.
func (api *QorAPI) GetReputationScore(validator string) (*ReputationScoreResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	rep, found := api.reputationKeeper.GetValidatorReputation(sdkCtx, validator)
	if !found {
		return nil, fmt.Errorf("validator reputation not found: %s", validator)
	}

	return &ReputationScoreResult{
		Validator:        validator,
		ReputationScore:  rep.CompositeScore,
		StakeComponent:   rep.StakeScore,
		PerfComponent:    rep.PerformanceScore,
		ContribComponent: rep.ContributionScore,
		TimeComponent:    rep.TimeScore,
	}, nil
}

// LayerInfoResult contains information about a multilayer chain layer.
type LayerInfoResult struct {
	LayerID     string `json:"layer_id"`
	LayerType   string `json:"layer_type"`
	Status      string `json:"status"`
	ChainID     string `json:"chain_id,omitempty"`
	Description string `json:"description"`
}

// GetLayerInfo returns information about a multilayer chain layer.
func (api *QorAPI) GetLayerInfo(layerID string) (*LayerInfoResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	layer, err := api.multilayerKeeper.GetLayer(sdkCtx, layerID)
	if err != nil {
		return nil, fmt.Errorf("layer not found: %s", layerID)
	}

	return &LayerInfoResult{
		LayerID:     layer.LayerID,
		LayerType:   string(layer.LayerType),
		Status:      string(layer.Status),
		ChainID:     layer.ChainID,
		Description: layer.Description,
	}, nil
}

// BridgeStatusResult contains bridge status for a remote chain.
type BridgeStatusResult struct {
	ChainID   string `json:"chain_id"`
	Connected bool   `json:"connected"`
	Status    string `json:"status"`
}

// GetBridgeStatus returns the bridge connection status for a remote chain.
func (api *QorAPI) GetBridgeStatus(chainID string) (*BridgeStatusResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	config, found := api.bridgeKeeper.GetChainConfig(sdkCtx, chainID)
	if !found {
		return &BridgeStatusResult{
			ChainID:   chainID,
			Connected: false,
			Status:    "not_configured",
		}, nil
	}

	return &BridgeStatusResult{
		ChainID:   config.ChainID,
		Connected: string(config.Status) == "active",
		Status:    string(config.Status),
	}, nil
}

// ---------------------------------------------------------------------------
// RL Consensus observability endpoints
// ---------------------------------------------------------------------------

// observationDimensionNames maps dimension indices to human-readable names.
var observationDimensionNames = [25]string{
	"block_utilization",
	"tx_count",
	"avg_tx_size",
	"block_time",
	"block_time_delta",
	"gas_price_50th",
	"gas_price_95th",
	"mempool_size",
	"mempool_bytes",
	"validator_count",
	"validator_gini",
	"missed_block_ratio",
	"avg_commit_latency",
	"max_commit_latency",
	"precommit_ratio",
	"failed_tx_ratio",
	"avg_gas_per_tx",
	"reward_per_validator",
	"slash_count",
	"jail_count",
	"inflation_rate",
	"bonded_ratio",
	"reputation_mean",
	"reputation_std_dev",
	"mev_estimate",
}

// RLAgentStatusResult contains the RL agent's current operational status.
type RLAgentStatusResult struct {
	AgentMode            string `json:"agent_mode"`
	CurrentEpoch         uint64 `json:"current_epoch"`
	IsActive             bool   `json:"is_active"`
	CircuitBreakerActive bool   `json:"circuit_breaker_active"`
}

// GetRLAgentStatus returns the current RL agent mode, epoch, and circuit breaker state.
func (api *QorAPI) GetRLAgentStatus() (*RLAgentStatusResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	status := api.rlconsensusKeeper.GetAgentStatus(sdkCtx)
	isActive := api.rlconsensusKeeper.IsRLActive(sdkCtx)

	return &RLAgentStatusResult{
		AgentMode:            status.Mode.String(),
		CurrentEpoch:         status.CurrentEpoch,
		IsActive:             isActive,
		CircuitBreakerActive: status.CircuitBreakerActive,
	}, nil
}

// RLObservationResult contains the latest observation vector with named dimensions.
type RLObservationResult struct {
	Height     int64             `json:"height"`
	Dimensions map[string]string `json:"dimensions"`
}

// GetRLObservation returns the latest RL observation vector as dimension name/value pairs.
func (api *QorAPI) GetRLObservation() (*RLObservationResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	obs, err := api.rlconsensusKeeper.GetLatestObservation(sdkCtx)
	if err != nil || obs == nil {
		return &RLObservationResult{
			Height:     0,
			Dimensions: map[string]string{},
		}, nil
	}

	dims := make(map[string]string, len(observationDimensionNames))
	for i, name := range observationDimensionNames {
		dims[name] = obs.Values[i]
	}

	return &RLObservationResult{
		Height:     obs.Height,
		Dimensions: dims,
	}, nil
}

// RLRewardResult contains the latest reward signal breakdown.
type RLRewardResult struct {
	Height                int64  `json:"height"`
	TotalReward           string `json:"total_reward"`
	ThroughputDelta       string `json:"throughput_delta"`
	FinalityDelta         string `json:"finality_delta"`
	DecentralizationDelta string `json:"decentralization_delta"`
	MEVEstimate           string `json:"mev_estimate"`
	FailedTxRatio         string `json:"failed_tx_ratio"`
}

// GetRLReward returns the latest reward signal with component breakdown.
func (api *QorAPI) GetRLReward() (*RLRewardResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	reward, err := api.rlconsensusKeeper.GetLatestReward(sdkCtx)
	if err != nil || reward == nil {
		return &RLRewardResult{}, nil
	}

	return &RLRewardResult{
		Height:                reward.Height,
		TotalReward:           reward.TotalReward,
		ThroughputDelta:       reward.ThroughputDelta,
		FinalityDelta:         reward.FinalityDelta,
		DecentralizationDelta: reward.DecentralizationDelta,
		MEVEstimate:           reward.MEVEstimate,
		FailedTxRatio:         reward.FailedTxRatio,
	}, nil
}

// PoolClassificationResult contains the validator pool assignment.
type PoolClassificationResult struct {
	Validator  string `json:"validator"`
	Pool       string `json:"pool"`
	AssignedAt int64  `json:"assigned_at"`
}

// GetPoolClassification returns the pool classification for a validator.
// This is a stub that will be wired to the QCA module in a future task.
func (api *QorAPI) GetPoolClassification(validator string) (*PoolClassificationResult, error) {
	return &PoolClassificationResult{
		Validator:  validator,
		Pool:       "unclassified",
		AssignedAt: 0,
	}, nil
}
