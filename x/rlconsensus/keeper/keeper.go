//go:build proprietary

package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// ReputationReader provides read access to the x/reputation module.
type ReputationReader interface {
	GetAllValidatorReputations(ctx sdk.Context) []ValidatorReputationData
	CalculateReputation(ctx sdk.Context, valAddr string) float64
}

// AIStatsReader provides read access to the x/ai module anomaly stats.
type AIStatsReader interface {
	GetAnomalyCount(ctx sdk.Context) uint64
}

// FeeMarketReader provides read access to the fee market base fee.
type FeeMarketReader interface {
	GetBaseFee(ctx sdk.Context) string // LegacyDec string
}

// ValidatorReputationData holds data extracted from x/reputation.
type ValidatorReputationData struct {
	Address        string
	CompositeScore float64
}

// Keeper manages the x/rlconsensus module state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	logger   log.Logger

	// Optional cross-module readers (set via setters after construction)
	reputationReader ReputationReader
	aiReader         AIStatsReader
	feeMarketReader  FeeMarketReader

	// MLP agent (initialized when policy weights are loaded)
	agent *PPOAgent
}

// NewKeeper creates a new rlconsensus keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
) *Keeper {
	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		logger:   logger.With("module", types.ModuleName),
	}
}

// SetReputationReader sets the reputation module reader.
func (k *Keeper) SetReputationReader(r ReputationReader) { k.reputationReader = r }

// SetAIReader sets the AI module stats reader.
func (k *Keeper) SetAIReader(r AIStatsReader) { k.aiReader = r }

// SetFeeMarketReader sets the fee market reader.
func (k *Keeper) SetFeeMarketReader(r FeeMarketReader) { k.feeMarketReader = r }

// Logger returns the keeper's logger.
func (k *Keeper) Logger() log.Logger { return k.logger }

// ---------------------------------------------------------------------------
// Params
// ---------------------------------------------------------------------------

// GetParams reads module parameters from the KVStore.
func (k *Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		k.logger.Error("failed to unmarshal rlconsensus params", "error", err)
		return types.DefaultParams()
	}
	return params
}

// SetParams writes module parameters to the KVStore.
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid rlconsensus params: %w", err)
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal rlconsensus params: %w", err)
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// AgentStatus
// ---------------------------------------------------------------------------

// GetAgentStatus reads the agent status from the KVStore.
func (k *Keeper) GetAgentStatus(ctx sdk.Context) types.AgentStatus {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AgentStatusKey)
	if bz == nil {
		return types.AgentStatus{Mode: types.AgentModeShadow}
	}
	var status types.AgentStatus
	if err := json.Unmarshal(bz, &status); err != nil {
		k.logger.Error("failed to unmarshal agent status", "error", err)
		return types.AgentStatus{Mode: types.AgentModeShadow}
	}
	return status
}

// SetAgentStatus writes the agent status to the KVStore.
func (k *Keeper) SetAgentStatus(ctx sdk.Context, status types.AgentStatus) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal agent status: %w", err)
	}
	store.Set(types.AgentStatusKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// PolicyWeights
// ---------------------------------------------------------------------------

// GetPolicyWeights reads the current policy weights from the KVStore.
func (k *Keeper) GetPolicyWeights(ctx sdk.Context) (*types.PolicyWeights, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PolicyWeightsKey)
	if bz == nil {
		return nil, nil
	}
	var pw types.PolicyWeights
	if err := json.Unmarshal(bz, &pw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy weights: %w", err)
	}
	return &pw, nil
}

// SetPolicyWeights writes policy weights to the KVStore.
func (k *Keeper) SetPolicyWeights(ctx sdk.Context, pw *types.PolicyWeights) error {
	if err := pw.Validate(); err != nil {
		return fmt.Errorf("invalid policy weights: %w", err)
	}
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(pw)
	if err != nil {
		return fmt.Errorf("failed to marshal policy weights: %w", err)
	}
	store.Set(types.PolicyWeightsKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// Observation (height-indexed)
// ---------------------------------------------------------------------------

// GetObservation reads the observation at a given height.
func (k *Keeper) GetObservation(ctx sdk.Context, height int64) (*types.Observation, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ObservationKey(height))
	if bz == nil {
		return nil, nil
	}
	var obs types.Observation
	if err := json.Unmarshal(bz, &obs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal observation at height %d: %w", height, err)
	}
	return &obs, nil
}

// SetObservation writes an observation to the KVStore keyed by its height.
func (k *Keeper) SetObservation(ctx sdk.Context, obs *types.Observation) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(obs)
	if err != nil {
		return fmt.Errorf("failed to marshal observation: %w", err)
	}
	store.Set(types.ObservationKey(obs.Height), bz)
	return nil
}

// GetLatestObservation scans backward from the current block height
// to find the most recent observation.
func (k *Keeper) GetLatestObservation(ctx sdk.Context) (*types.Observation, error) {
	store := ctx.KVStore(k.storeKey)

	// Use a reverse iterator over the observation prefix to find the latest entry.
	iter := storetypes.KVStoreReversePrefixIterator(store, types.ObservationKeyPrefix)
	defer iter.Close()

	if !iter.Valid() {
		return nil, nil
	}

	var obs types.Observation
	if err := json.Unmarshal(iter.Value(), &obs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal latest observation: %w", err)
	}
	return &obs, nil
}

// ---------------------------------------------------------------------------
// Reward (height-indexed)
// ---------------------------------------------------------------------------

// GetReward reads the reward record at a given height.
func (k *Keeper) GetReward(ctx sdk.Context, height int64) (*types.Reward, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RewardKey(height))
	if bz == nil {
		return nil, nil
	}
	var reward types.Reward
	if err := json.Unmarshal(bz, &reward); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reward at height %d: %w", height, err)
	}
	return &reward, nil
}

// SetReward writes a reward record to the KVStore keyed by its height.
func (k *Keeper) SetReward(ctx sdk.Context, reward *types.Reward) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(reward)
	if err != nil {
		return fmt.Errorf("failed to marshal reward: %w", err)
	}
	store.Set(types.RewardKey(reward.Height), bz)
	return nil
}

// GetLatestReward scans backward from the current block height
// to find the most recent reward record.
func (k *Keeper) GetLatestReward(ctx sdk.Context) (*types.Reward, error) {
	store := ctx.KVStore(k.storeKey)

	iter := storetypes.KVStoreReversePrefixIterator(store, types.RewardKeyPrefix)
	defer iter.Close()

	if !iter.Valid() {
		return nil, nil
	}

	var reward types.Reward
	if err := json.Unmarshal(iter.Value(), &reward); err != nil {
		return nil, fmt.Errorf("failed to unmarshal latest reward: %w", err)
	}
	return &reward, nil
}

// ---------------------------------------------------------------------------
// AppliedConsensusParams
// ---------------------------------------------------------------------------

// GetAppliedParams reads the most recently applied consensus parameters.
func (k *Keeper) GetAppliedParams(ctx sdk.Context) AppliedConsensusParams {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AppliedParamsKey)
	if bz == nil {
		return DefaultAppliedParams()
	}
	var p AppliedConsensusParams
	if err := json.Unmarshal(bz, &p); err != nil {
		k.logger.Error("failed to unmarshal applied params", "error", err)
		return DefaultAppliedParams()
	}
	return p
}

// SetAppliedParams writes applied consensus parameters to the KVStore.
func (k *Keeper) SetAppliedParams(ctx sdk.Context, p AppliedConsensusParams) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal applied params: %w", err)
	}
	store.Set(types.AppliedParamsKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// CircuitBreakerState (stored)
// ---------------------------------------------------------------------------

// GetCircuitBreakerState reads the circuit breaker state from the KVStore.
func (k *Keeper) GetCircuitBreakerState(ctx sdk.Context) CircuitBreakerState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CircuitBreakerStateKey)
	if bz == nil {
		return CircuitBreakerState{}
	}
	var state CircuitBreakerState
	if err := json.Unmarshal(bz, &state); err != nil {
		return CircuitBreakerState{}
	}
	return state
}

// SetCircuitBreakerState writes the circuit breaker state.
func (k *Keeper) SetCircuitBreakerState(ctx sdk.Context, state CircuitBreakerState) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal circuit breaker state: %w", err)
	}
	store.Set(types.CircuitBreakerStateKey, bz)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// heightToBytes encodes a block height as big-endian 8 bytes.
func heightToBytes(h int64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(h))
	return bz
}
