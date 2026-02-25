//go:build proprietary

package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
	"github.com/qorechain/qorechain-core/x/qca/types"
)

func setupBondingTestKeeper(t *testing.T) (Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	k := NewKeeper(
		codec.NewProtoCodec(nil),
		storeKey,
		reputationkeeper.Keeper{},
		NewHeuristicSelector(),
		log.NewNopLogger(),
	)

	// Set default config
	if err := k.SetConfig(ctx, types.DefaultQCAConfig()); err != nil {
		t.Fatal(err)
	}

	return k, ctx
}

// decStr is a test helper that builds a LegacyDec from a string.
func decStr(s string) math.LegacyDec {
	return math.LegacyMustNewDecFromStr(s)
}

// ---------------------------------------------------------------------------
// Test 1: Zero stake returns zero reward
// ---------------------------------------------------------------------------

func TestBondingReward_ZeroStake(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	reward, err := k.CalculateBondingReward(ctx, 0, 1000, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reward.IsZero() {
		t.Fatalf("expected zero reward for zero stake, got %s", reward)
	}
}

// ---------------------------------------------------------------------------
// Test 2: Zero loyalty -- loyaltyFactor should be 1 (since log(1+0)=0)
// ---------------------------------------------------------------------------

func TestBondingReward_ZeroLoyalty(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	// S_v=1000000 (1 QOR), L_v=0, rep=0.5 (neutral)
	// With defaults: alpha=0.1, beta=1.0, phase=1.5
	// Q(0.5) = 1 + 0.5*(0.5-0.5) = 1.0
	// loyaltyFactor = 1 + 0.1*log(1+0) = 1 + 0.1*0 = 1.0
	// R = 1.0 * 1000000 * 1.0 * 1.0 * 1.5 = 1500000
	reward, err := k.CalculateBondingReward(ctx, 1_000_000, 0, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decStr("1500000")
	diff := reward.Sub(expected).Abs()
	if diff.GT(decStr("0.01")) {
		t.Fatalf("zero loyalty: expected %s, got %s (diff %s)", expected, reward, diff)
	}
}

// ---------------------------------------------------------------------------
// Test 3: Loyalty increases reward
// ---------------------------------------------------------------------------

func TestBondingReward_LoyaltyIncreasesReward(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	// Same stake and reputation, different loyalty
	rewardNoLoyalty, err := k.CalculateBondingReward(ctx, 1_000_000, 0, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rewardWithLoyalty, err := k.CalculateBondingReward(ctx, 1_000_000, 1000, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !rewardWithLoyalty.GT(rewardNoLoyalty) {
		t.Fatalf("loyalty should increase reward: no loyalty=%s, with loyalty=%s",
			rewardNoLoyalty, rewardWithLoyalty)
	}
}

// ---------------------------------------------------------------------------
// Test 4: Higher reputation increases reward
// ---------------------------------------------------------------------------

func TestBondingReward_HigherReputationIncreasesReward(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	rewardLowRep, err := k.CalculateBondingReward(ctx, 1_000_000, 100, 0.1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rewardHighRep, err := k.CalculateBondingReward(ctx, 1_000_000, 100, 0.9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !rewardHighRep.GT(rewardLowRep) {
		t.Fatalf("higher reputation should increase reward: low rep=%s, high rep=%s",
			rewardLowRep, rewardHighRep)
	}
}

// ---------------------------------------------------------------------------
// Test 5: Q clamping -- extreme reputation scores clamp Q to [0.75, 1.25]
// ---------------------------------------------------------------------------

func TestBondingReward_QClampingLow(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	// rep=-1.0 is extreme negative. Q = 1 + 0.5*(-1-0.5) = 1 - 0.75 = 0.25
	// Should be clamped to 0.75.
	// With S_v=1000000, L_v=0, beta=1, phase=1.5:
	// R = 1.0 * 1000000 * 1.0 * 0.75 * 1.5 = 1125000
	reward, err := k.CalculateBondingReward(ctx, 1_000_000, 0, -1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decStr("1125000")
	diff := reward.Sub(expected).Abs()
	if diff.GT(decStr("0.01")) {
		t.Fatalf("Q clamping low: expected %s, got %s (diff %s)", expected, reward, diff)
	}
}

func TestBondingReward_QClampingHigh(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	// rep=5.0 is extreme positive. Q = 1 + 0.5*(5.0-0.5) = 1 + 2.25 = 3.25
	// Should be clamped to 1.25.
	// With S_v=1000000, L_v=0, beta=1, phase=1.5:
	// R = 1.0 * 1000000 * 1.0 * 1.25 * 1.5 = 1875000
	reward, err := k.CalculateBondingReward(ctx, 1_000_000, 0, 5.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decStr("1875000")
	diff := reward.Sub(expected).Abs()
	if diff.GT(decStr("0.01")) {
		t.Fatalf("Q clamping high: expected %s, got %s (diff %s)", expected, reward, diff)
	}
}

// ---------------------------------------------------------------------------
// Test 6: Stats counter increments
// ---------------------------------------------------------------------------

func TestBondingReward_StatsIncrement(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	statsBefore := k.GetStats(ctx)
	if statsBefore.BondingCalculations != 0 {
		t.Fatalf("expected 0 bonding calculations initially, got %d", statsBefore.BondingCalculations)
	}

	_, err := k.CalculateBondingReward(ctx, 1_000_000, 100, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statsAfter := k.GetStats(ctx)
	if statsAfter.BondingCalculations != 1 {
		t.Fatalf("expected 1 bonding calculation after call, got %d", statsAfter.BondingCalculations)
	}
}

// ---------------------------------------------------------------------------
// Test 7: Neutral reputation gives Q = 1.0
// ---------------------------------------------------------------------------

func TestBondingReward_NeutralReputation(t *testing.T) {
	k, ctx := setupBondingTestKeeper(t)

	// rep=0.5 -> Q = 1 + 0.5*(0.5-0.5) = 1.0
	// S_v=2000000, L_v=0, beta=1, phase=1.5
	// R = 1.0 * 2000000 * 1.0 * 1.0 * 1.5 = 3000000
	reward, err := k.CalculateBondingReward(ctx, 2_000_000, 0, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decStr("3000000")
	diff := reward.Sub(expected).Abs()
	if diff.GT(decStr("0.01")) {
		t.Fatalf("neutral rep: expected %s, got %s (diff %s)", expected, reward, diff)
	}
}
