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

	abstractaccountmod "github.com/qorechain/qorechain-core/x/abstractaccount"
	aimod "github.com/qorechain/qorechain-core/x/ai"
	babylonmod "github.com/qorechain/qorechain-core/x/babylon"
	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	burnmod "github.com/qorechain/qorechain-core/x/burn"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	fairblockmod "github.com/qorechain/qorechain-core/x/fairblock"
	gasabstractionmod "github.com/qorechain/qorechain-core/x/gasabstraction"
	rdkmod "github.com/qorechain/qorechain-core/x/rdk"
	rdktypes "github.com/qorechain/qorechain-core/x/rdk/types"
	inflationmod "github.com/qorechain/qorechain-core/x/inflation"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	xqoremod "github.com/qorechain/qorechain-core/x/xqore"
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
	burnKeeper             burnmod.BurnKeeper
	xqoreKeeper            xqoremod.XQOREKeeper
	inflationKeeper        inflationmod.InflationKeeper
	babylonKeeper          babylonmod.BabylonKeeper
	abstractAccountKeeper  abstractaccountmod.AbstractAccountKeeper
	fairBlockKeeper        fairblockmod.FairBlockKeeper
	gasAbstractionKeeper   gasabstractionmod.GasAbstractionKeeper
	rdkKeeper              rdkmod.RDKKeeper
	laneConfig             []LaneConfigResult
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
	burnKeeper burnmod.BurnKeeper,
	xqoreKeeper xqoremod.XQOREKeeper,
	inflationKeeper inflationmod.InflationKeeper,
	babylonKeeper babylonmod.BabylonKeeper,
	abstractAccountKeeper abstractaccountmod.AbstractAccountKeeper,
	fairBlockKeeper fairblockmod.FairBlockKeeper,
	gasAbstractionKeeper gasabstractionmod.GasAbstractionKeeper,
	rdkKeeper rdkmod.RDKKeeper,
	laneConfig []LaneConfigResult,
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
		burnKeeper:            burnKeeper,
		xqoreKeeper:           xqoreKeeper,
		inflationKeeper:       inflationKeeper,
		babylonKeeper:         babylonKeeper,
		abstractAccountKeeper: abstractAccountKeeper,
		fairBlockKeeper:       fairBlockKeeper,
		gasAbstractionKeeper:  gasAbstractionKeeper,
		rdkKeeper:              rdkKeeper,
		laneConfig:            laneConfig,
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

// HybridSignatureModeResult contains the current hybrid signature mode.
type HybridSignatureModeResult struct {
	Mode        uint32 `json:"mode"`
	ModeString  string `json:"mode_string"`
	Description string `json:"description"`
}

// GetHybridSignatureMode returns the current PQC hybrid signature mode.
func (api *QorAPI) GetHybridSignatureMode() (*HybridSignatureModeResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	mode := api.pqcKeeper.GetHybridSignatureMode(sdkCtx)

	return &HybridSignatureModeResult{
		Mode:        uint32(mode),
		ModeString:  mode.String(),
		Description: mode.Description(),
	}, nil
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

// ---------------------------------------------------------------------------
// Tokenomics endpoints (burn, xQORE, inflation)
// ---------------------------------------------------------------------------

// BurnStatsResult contains burn module statistics.
type BurnStatsResult struct {
	TotalBurned    string            `json:"total_burned"`
	BurnsBySource  map[string]string `json:"burns_by_source"`
	LastBurnHeight int64             `json:"last_burn_height"`
}

// GetBurnStats returns burn statistics.
func (api *QorAPI) GetBurnStats() (*BurnStatsResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	stats := api.burnKeeper.GetBurnStats(sdkCtx)

	bySource := make(map[string]string, len(stats.BurnsBySource))
	for source, amount := range stats.BurnsBySource {
		bySource[string(source)] = amount.String()
	}

	return &BurnStatsResult{
		TotalBurned:    stats.TotalBurned.String(),
		BurnsBySource:  bySource,
		LastBurnHeight: stats.LastBurnHeight,
	}, nil
}

// XQOREPositionResult contains the xQORE position for an address.
type XQOREPositionResult struct {
	Address    string `json:"address"`
	Found      bool   `json:"found"`
	Locked     string `json:"locked,omitempty"`
	XBalance   string `json:"x_balance,omitempty"`
	LockHeight int64  `json:"lock_height,omitempty"`
	LockTime   string `json:"lock_time,omitempty"`
}

// GetXQOREPosition returns the xQORE position for an address.
func (api *QorAPI) GetXQOREPosition(address string) (*XQOREPositionResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	pos, found := api.xqoreKeeper.GetPosition(sdkCtx, addr)
	if !found {
		return &XQOREPositionResult{
			Address: address,
			Found:   false,
		}, nil
	}

	return &XQOREPositionResult{
		Address:    address,
		Found:      true,
		Locked:     pos.Locked.String(),
		XBalance:   pos.XBalance.String(),
		LockHeight: pos.LockHeight,
		LockTime:   pos.LockTime.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

// InflationRateResult contains the current inflation rate and epoch info.
type InflationRateResult struct {
	CurrentRate  string `json:"current_rate"`
	CurrentEpoch uint64 `json:"current_epoch"`
	CurrentYear  uint64 `json:"current_year"`
	TotalMinted  string `json:"total_minted"`
}

// GetInflationRate returns the current inflation rate and epoch information.
func (api *QorAPI) GetInflationRate() (*InflationRateResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	rate := api.inflationKeeper.GetCurrentInflationRate(sdkCtx)
	epoch := api.inflationKeeper.GetEpochInfo(sdkCtx)

	return &InflationRateResult{
		CurrentRate:  rate.String(),
		CurrentEpoch: epoch.CurrentEpoch,
		CurrentYear:  epoch.CurrentYear,
		TotalMinted:  epoch.TotalMinted.String(),
	}, nil
}

// TokenomicsOverviewResult contains combined tokenomics statistics.
type TokenomicsOverviewResult struct {
	TotalBurned    string `json:"total_burned"`
	TotalLocked    string `json:"total_locked"`
	TotalXQORE     string `json:"total_xqore"`
	InflationRate  string `json:"inflation_rate"`
	TotalMinted    string `json:"total_minted"`
}

// GetTokenomicsOverview returns a combined view of all tokenomics stats.
func (api *QorAPI) GetTokenomicsOverview() (*TokenomicsOverviewResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	stats := api.burnKeeper.GetBurnStats(sdkCtx)
	totalLocked := api.xqoreKeeper.GetTotalLocked(sdkCtx)
	totalXQORE := api.xqoreKeeper.GetTotalXQORESupply(sdkCtx)
	rate := api.inflationKeeper.GetCurrentInflationRate(sdkCtx)
	epoch := api.inflationKeeper.GetEpochInfo(sdkCtx)

	return &TokenomicsOverviewResult{
		TotalBurned:   stats.TotalBurned.String(),
		TotalLocked:   totalLocked.String(),
		TotalXQORE:    totalXQORE.String(),
		InflationRate: rate.String(),
		TotalMinted:   epoch.TotalMinted.String(),
	}, nil
}

// ---------------------------------------------------------------------------
// v1.2.0 endpoints: Babylon, AbstractAccount, FairBlock, GasAbstraction, Lanes
// ---------------------------------------------------------------------------

// BTCStakingPositionResult contains a BTC restaking position.
type BTCStakingPositionResult struct {
	StakerAddress string `json:"staker_address"`
	Found         bool   `json:"found"`
	BTCAmount     string `json:"btc_amount,omitempty"`
	Status        string `json:"status,omitempty"`
	StakeHeight   int64  `json:"stake_height,omitempty"`
}

// GetBTCStakingPosition returns the BTC restaking position for an address.
func (api *QorAPI) GetBTCStakingPosition(address string) (*BTCStakingPositionResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	pos, found := api.babylonKeeper.GetStakingPosition(sdkCtx, address)
	if !found {
		return &BTCStakingPositionResult{
			StakerAddress: address,
			Found:         false,
		}, nil
	}

	return &BTCStakingPositionResult{
		StakerAddress: address,
		Found:         true,
		BTCAmount:     pos.BTCAmount,
		Status:        pos.Status,
		StakeHeight:   pos.StakeHeight,
	}, nil
}

// AbstractAccountResult contains abstract account information.
type AbstractAccountResult struct {
	Address     string `json:"address"`
	Found       bool   `json:"found"`
	AccountType string `json:"account_type,omitempty"`
	Contract    string `json:"contract_address,omitempty"`
}

// GetAbstractAccount returns the abstract account info for an address.
func (api *QorAPI) GetAbstractAccount(address string) (*AbstractAccountResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	acct, found := api.abstractAccountKeeper.GetAccount(sdkCtx, address)
	if !found {
		return &AbstractAccountResult{
			Address: address,
			Found:   false,
		}, nil
	}

	return &AbstractAccountResult{
		Address:     address,
		Found:       true,
		AccountType: acct.AccountType,
		Contract:    acct.ContractAddress,
	}, nil
}

// FairBlockStatusResult contains FairBlock module status.
type FairBlockStatusResult struct {
	Enabled        bool   `json:"enabled"`
	TIBEThreshold  uint32 `json:"tibe_threshold"`
	DecryptionDelay int64 `json:"decryption_delay"`
}

// GetFairBlockStatus returns the FairBlock module's current configuration.
func (api *QorAPI) GetFairBlockStatus() (*FairBlockStatusResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	config := api.fairBlockKeeper.GetConfig(sdkCtx)

	return &FairBlockStatusResult{
		Enabled:         config.Enabled,
		TIBEThreshold:   config.TIBEThreshold,
		DecryptionDelay: config.DecryptionDelay,
	}, nil
}

// GasAbstractionConfigResult contains gas abstraction configuration.
type GasAbstractionConfigResult struct {
	Enabled        bool                `json:"enabled"`
	NativeDenom    string              `json:"native_denom"`
	AcceptedTokens []AcceptedTokenInfo `json:"accepted_tokens"`
}

// AcceptedTokenInfo contains info about an accepted fee token.
type AcceptedTokenInfo struct {
	Denom          string `json:"denom"`
	ConversionRate string `json:"conversion_rate"`
}

// GetGasAbstractionConfig returns the gas abstraction module configuration.
func (api *QorAPI) GetGasAbstractionConfig() (*GasAbstractionConfigResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	config := api.gasAbstractionKeeper.GetConfig(sdkCtx)

	tokens := make([]AcceptedTokenInfo, len(config.AcceptedTokens))
	for i, t := range config.AcceptedTokens {
		tokens[i] = AcceptedTokenInfo{
			Denom:          t.Denom,
			ConversionRate: fmt.Sprintf("%.6f", t.ConversionRate),
		}
	}

	return &GasAbstractionConfigResult{
		Enabled:        config.Enabled,
		NativeDenom:    config.NativeDenom,
		AcceptedTokens: tokens,
	}, nil
}

// LaneConfigResult contains lane configuration info.
type LaneConfigResult struct {
	Name          string  `json:"name"`
	Priority      int     `json:"priority"`
	MaxBlockSpace float64 `json:"max_block_space"`
	Description   string  `json:"description"`
}

// GetLaneConfiguration returns the current 5-lane tx prioritization configuration.
// Lane config is compile-time static and set via the laneConfig field.
func (api *QorAPI) GetLaneConfiguration() ([]LaneConfigResult, error) {
	return api.laneConfig, nil
}

// --- v1.3.0 RDK Endpoints ---

// RollupStatusResult contains rollup dashboard info.
type RollupStatusResult struct {
	RollupID       string `json:"rollup_id"`
	Creator        string `json:"creator"`
	Profile        string `json:"profile"`
	SettlementMode string `json:"settlement_mode"`
	DABackend      string `json:"da_backend"`
	Status         string `json:"status"`
	BlockTimeMs    uint64 `json:"block_time_ms"`
	MaxTxPerBlock  uint64 `json:"max_tx_per_block"`
	VMType         string `json:"vm_type"`
	StakeAmount    int64  `json:"stake_amount"`
	LatestBatch    int64  `json:"latest_batch_index"`
	CreatedHeight  int64  `json:"created_height"`
}

// GetRollupStatus returns rollup dashboard data.
func (api *QorAPI) GetRollupStatus(rollupID string) (*RollupStatusResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	rollup, err := api.rdkKeeper.GetRollup(sdkCtx, rollupID)
	if err != nil {
		return nil, err
	}

	var latestBatchIdx int64
	if batch, bErr := api.rdkKeeper.GetLatestBatch(sdkCtx, rollupID); bErr == nil {
		latestBatchIdx = int64(batch.BatchIndex)
	}

	return &RollupStatusResult{
		RollupID:       rollup.RollupID,
		Creator:        rollup.Creator,
		Profile:        string(rollup.Profile),
		SettlementMode: string(rollup.SettlementMode),
		DABackend:      string(rollup.DABackend),
		Status:         string(rollup.Status),
		BlockTimeMs:    rollup.BlockTimeMs,
		MaxTxPerBlock:  rollup.MaxTxPerBlock,
		VMType:         rollup.VMType,
		StakeAmount:    rollup.StakeAmount,
		LatestBatch:    latestBatchIdx,
		CreatedHeight:  rollup.CreatedHeight,
	}, nil
}

// RollupListResult contains a list of rollups.
type RollupListResult struct {
	Rollups []RollupStatusResult `json:"rollups"`
	Total   int                  `json:"total"`
}

// ListRollups returns all rollups, optionally filtered by creator.
func (api *QorAPI) ListRollups(creator string) (*RollupListResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	var err error
	var rollupConfigs []*rdktypes.RollupConfig

	if creator != "" {
		rollupConfigs, err = api.rdkKeeper.ListRollupsByCreator(sdkCtx, creator)
	} else {
		rollupConfigs, err = api.rdkKeeper.ListRollups(sdkCtx)
	}
	if err != nil {
		return nil, err
	}

	results := make([]RollupStatusResult, 0, len(rollupConfigs))
	for _, rc := range rollupConfigs {
		results = append(results, RollupStatusResult{
			RollupID:       rc.RollupID,
			Creator:        rc.Creator,
			Profile:        string(rc.Profile),
			SettlementMode: string(rc.SettlementMode),
			DABackend:      string(rc.DABackend),
			Status:         string(rc.Status),
			BlockTimeMs:    rc.BlockTimeMs,
			MaxTxPerBlock:  rc.MaxTxPerBlock,
			VMType:         rc.VMType,
			StakeAmount:    rc.StakeAmount,
			CreatedHeight:  rc.CreatedHeight,
		})
	}

	return &RollupListResult{
		Rollups: results,
		Total:   len(results),
	}, nil
}

// SettlementBatchResult contains batch info.
type SettlementBatchResult struct {
	RollupID    string `json:"rollup_id"`
	BatchIndex  uint64 `json:"batch_index"`
	TxCount     uint64 `json:"tx_count"`
	ProofType   string `json:"proof_type"`
	Status      string `json:"status"`
	SubmittedAt int64  `json:"submitted_at"`
	FinalizedAt int64  `json:"finalized_at"`
}

// GetSettlementBatch returns info about a specific or latest batch.
func (api *QorAPI) GetSettlementBatch(rollupID string, batchIndex int64) (*SettlementBatchResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)

	var batch *rdktypes.SettlementBatch
	var err error
	if batchIndex >= 0 {
		batch, err = api.rdkKeeper.GetBatch(sdkCtx, rollupID, uint64(batchIndex))
	} else {
		batch, err = api.rdkKeeper.GetLatestBatch(sdkCtx, rollupID)
	}
	if err != nil {
		return nil, err
	}

	return &SettlementBatchResult{
		RollupID:    batch.RollupID,
		BatchIndex:  batch.BatchIndex,
		TxCount:     batch.TxCount,
		ProofType:   string(batch.ProofType),
		Status:      string(batch.Status),
		SubmittedAt: batch.SubmittedAt,
		FinalizedAt: batch.FinalizedAt,
	}, nil
}

// SuggestProfileResult contains a suggested profile.
type SuggestProfileResult struct {
	UseCase          string `json:"use_case"`
	SuggestedProfile string `json:"suggested_profile"`
}

// SuggestRollupProfile returns an AI-suggested rollup profile for a use case.
func (api *QorAPI) SuggestRollupProfile(useCase string) (*SuggestProfileResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	profile, err := api.rdkKeeper.SuggestProfile(sdkCtx, useCase)
	if err != nil {
		return nil, err
	}
	return &SuggestProfileResult{
		UseCase:          useCase,
		SuggestedProfile: string(*profile),
	}, nil
}

// DABlobStatusResult contains DA blob status.
type DABlobStatusResult struct {
	RollupID  string `json:"rollup_id"`
	BlobIndex uint64 `json:"blob_index"`
	Size      int    `json:"size"`
	Pruned    bool   `json:"pruned"`
	Height    int64  `json:"height"`
}

// GetDABlobStatus returns the DA blob storage status.
func (api *QorAPI) GetDABlobStatus(rollupID string, blobIndex int64) (*DABlobStatusResult, error) {
	sdkCtx := sdk.UnwrapSDKContext(api.ctx)
	blob, err := api.rdkKeeper.GetDABlob(sdkCtx, rollupID, uint64(blobIndex))
	if err != nil {
		return nil, err
	}
	return &DABlobStatusResult{
		RollupID:  blob.RollupID,
		BlobIndex: blob.BlobIndex,
		Size:      len(blob.Data),
		Pruned:    blob.Pruned,
		Height:    blob.Height,
	}, nil
}
