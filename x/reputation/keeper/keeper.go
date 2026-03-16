package keeper

import (
	"encoding/json"
	"fmt"
	"math"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// Keeper manages the x/reputation module state.
type Keeper struct {
	cdc           codec.Codec
	storeKey      storetypes.StoreKey
	stakingKeeper *stakingkeeper.Keeper
	logger        log.Logger
}

// NewKeeper creates a new reputation keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	stakingKeeper *stakingkeeper.Keeper,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		stakingKeeper: stakingKeeper,
		logger:        logger.With("module", types.ModuleName),
	}
}

func (k Keeper) Logger() log.Logger { return k.logger }

// ---- Params ----

func (k Keeper) GetParams(ctx sdk.Context) types.ReputationParams {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultReputationParams()
	}
	var params types.ReputationParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultReputationParams()
	}
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.ReputationParams) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// ---- Validator Reputation ----

func validatorKey(address string) []byte {
	return append(types.ValidatorPrefix, []byte(address)...)
}

func (k Keeper) GetValidatorReputation(ctx sdk.Context, address string) (types.ValidatorReputation, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(validatorKey(address))
	if bz == nil {
		return types.ValidatorReputation{}, false
	}
	var rep types.ValidatorReputation
	if err := json.Unmarshal(bz, &rep); err != nil {
		return types.ValidatorReputation{}, false
	}
	return rep, true
}

func (k Keeper) SetValidatorReputation(ctx sdk.Context, rep types.ValidatorReputation) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(rep)
	store.Set(validatorKey(rep.Address), bz)
}

func (k Keeper) GetAllValidatorReputations(ctx sdk.Context) []types.ValidatorReputation {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ValidatorPrefix)
	defer iter.Close()

	var reps []types.ValidatorReputation
	for ; iter.Valid(); iter.Next() {
		var rep types.ValidatorReputation
		if err := json.Unmarshal(iter.Value(), &rep); err != nil {
			continue
		}
		reps = append(reps, rep)
	}
	return reps
}

// ---- Scoring Logic ----

// CalculateReputation computes the composite reputation score per the whitepaper formula:
// R_i = α·S_i + β·P_i + γ·C_i + δ·T_i with time decay.
func (k Keeper) CalculateReputation(ctx sdk.Context, valAddr string) float64 {
	params := k.GetParams(ctx)
	return k.calculateReputationWithParams(ctx, valAddr, params)
}

// calculateReputationWithParams avoids redundant GetParams calls when computing
// scores in batch (e.g., during EndBlocker). It fetches the reputation from store.
func (k Keeper) calculateReputationWithParams(ctx sdk.Context, valAddr string, params types.ReputationParams) float64 {
	rep, found := k.GetValidatorReputation(ctx, valAddr)
	if !found {
		return params.MinScore
	}
	return k.calculateReputationFromExisting(ctx, rep, params).CompositeScore
}

// calculateReputationFromExisting computes the composite reputation score from
// an already-loaded ValidatorReputation, avoiding a redundant store read.
func (k Keeper) calculateReputationFromExisting(ctx sdk.Context, rep types.ValidatorReputation, params types.ReputationParams) types.ValidatorReputation {
	// Component scores (each normalized to 0.0-1.0)
	S := rep.StakeScore // Use stored stake score directly
	P := k.calculatePerformanceScore(rep)
	C := k.calculateContributionScore(rep)
	T := k.calculateTimeScore(ctx, rep)

	// Composite: R_i = α·S_i + β·P_i + γ·C_i + δ·T_i
	composite := params.Alpha*S + params.Beta*P + params.Gamma*C + params.Delta*T

	// Time decay: R_new = R_old * exp(-Δt/λ) + R_calc * (1 - exp(-Δt/λ))
	elapsed := float64(ctx.BlockHeight() - rep.LastUpdatedHeight)
	if elapsed < 0 {
		elapsed = 0
	}
	decayFactor := math.Exp(-elapsed / params.Lambda)

	smoothed := rep.CompositeScore*decayFactor + composite*(1.0-decayFactor)

	// Enforce minimum
	if smoothed < params.MinScore {
		smoothed = params.MinScore
	}

	rep.StakeScore = S
	rep.PerformanceScore = P
	rep.ContributionScore = C
	rep.TimeScore = T
	rep.CompositeScore = smoothed
	rep.LastUpdatedHeight = ctx.BlockHeight()

	return rep
}

func (k Keeper) calculatePerformanceScore(rep types.ValidatorReputation) float64 {
	// Performance = (uptime_blocks - missed_blocks) / (uptime_blocks + missed_blocks)
	total := rep.UptimeBlocks + rep.MissedBlocks
	if total == 0 {
		return 0.5 // Default for new validators
	}
	return float64(rep.UptimeBlocks) / float64(total)
}

func (k Keeper) calculateContributionScore(rep types.ValidatorReputation) float64 {
	// Contribution based on community votes, capped at 1.0
	if rep.CommunityVotes <= 0 {
		return 0.0
	}
	// Logarithmic scale: more votes = diminishing returns
	return math.Min(math.Log1p(float64(rep.CommunityVotes))/5.0, 1.0)
}

func (k Keeper) calculateTimeScore(ctx sdk.Context, rep types.ValidatorReputation) float64 {
	// Longevity bonus: longer participation = higher score
	age := float64(ctx.BlockHeight() - rep.JoinedAtHeight)
	if age <= 0 {
		return 0.0
	}
	// Asymptotic approach to 1.0 over ~10,000 blocks
	return 1.0 - math.Exp(-age/10000.0)
}

// ---- EndBlocker ----

// EndBlocker runs at the end of each block to update validator performance.
func (k Keeper) EndBlocker(ctx sdk.Context) error {
	// Get the current block's proposer
	proposer := ctx.BlockHeader().ProposerAddress
	if proposer == nil {
		return nil
	}

	proposerAddr := sdk.ConsAddress(proposer).String()

	// Read params once for the entire batch instead of per-validator.
	params := k.GetParams(ctx)

	// Update all validator reputations
	reps := k.GetAllValidatorReputations(ctx)
	for _, rep := range reps {
		// Increment uptime for all registered validators.
		// TODO: Check actual block signatures to only credit signing validators.
		rep.UptimeBlocks++

		// If this validator was the proposer, increment proposed blocks
		if rep.Address == proposerAddr {
			rep.ProposedBlocks++
		}

		// Recalculate composite score using the already-loaded rep (no double read).
		rep = k.calculateReputationFromExisting(ctx, rep, params)

		k.SetValidatorReputation(ctx, rep)

		// Record historical score for this block.
		k.RecordHistoricalScore(ctx, rep.Address, rep.CompositeScore)
	}

	return nil
}

// ---- Historical Scores ----

// RecordHistoricalScore persists a validator's composite score at the current
// block height. The key uses zero-padded height for correct lexicographic ordering.
func (k Keeper) RecordHistoricalScore(ctx sdk.Context, valAddr string, score float64) {
	key := append([]byte{}, types.HistoryPrefix...)
	key = append(key, []byte(fmt.Sprintf("%s/%020d", valAddr, ctx.BlockHeight()))...)
	hs := types.HistoricalScore{
		Height:    ctx.BlockHeight(),
		Score:     score,
		Timestamp: ctx.BlockTime(),
	}
	bz, _ := json.Marshal(hs)
	store := ctx.KVStore(k.storeKey)
	store.Set(key, bz)
}

// ---- Genesis ----

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set reputation params: %v", err))
	}
	for _, val := range gs.Validators {
		k.SetValidatorReputation(ctx, val)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		Validators: k.GetAllValidatorReputations(ctx),
	}
}
