//go:build proprietary

package keeper

import (
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
)

func setupQDRWTestHandler(t *testing.T) (*QDRWTallyHandler, Keeper, sdk.Context) {
	t.Helper()

	k, ctx := setupBondingTestKeeper(t)

	handler := NewQDRWTallyHandler(&k, rlconsensusmod.NilTokenomicsKeeper{})

	return handler, k, ctx
}

// ---------------------------------------------------------------------------
// Test 1: QDRW disabled returns raw stake
// ---------------------------------------------------------------------------

func TestQDRW_DisabledReturnsRawStake(t *testing.T) {
	handler, k, ctx := setupQDRWTestHandler(t)

	// Ensure QDRW is disabled (default)
	cfg := k.GetConfig(ctx)
	if cfg.QDRWConfig.Enabled {
		t.Fatal("expected QDRW to be disabled by default")
	}

	voter := sdk.AccAddress([]byte("voter1______________"))
	stakedAmount := uint64(1_000_000)

	vp, err := handler.CalculateVotingPower(ctx, voter, stakedAmount, 0.9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := math.LegacyNewDec(int64(stakedAmount))
	if !vp.Equal(expected) {
		t.Fatalf("QDRW disabled: expected VP=%s, got %s", expected, vp)
	}
}

// ---------------------------------------------------------------------------
// Test 2: Quadratic dampening -- whale with 100x stake gets ~10x VP, not 100x
// ---------------------------------------------------------------------------

func TestQDRW_QuadraticDampening(t *testing.T) {
	handler, k, ctx := setupQDRWTestHandler(t)

	// Enable QDRW
	cfg := k.GetConfig(ctx)
	cfg.QDRWConfig.Enabled = true
	if err := k.SetConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	voter := sdk.AccAddress([]byte("voter1______________"))
	neutralRep := 0.5

	// Voter A: 1M uqor
	vpA, err := handler.CalculateVotingPower(ctx, voter, 1_000_000, neutralRep)
	if err != nil {
		t.Fatalf("unexpected error for voter A: %v", err)
	}

	// Voter B: 100M uqor (100x more stake)
	vpB, err := handler.CalculateVotingPower(ctx, voter, 100_000_000, neutralRep)
	if err != nil {
		t.Fatalf("unexpected error for voter B: %v", err)
	}

	// With same reputation, the ratio should be approximately sqrt(100) = 10x
	// VP = sqrt(stake) * repMultiplier => ratio = sqrt(100M)/sqrt(1M) = sqrt(100) = 10
	if vpA.IsZero() {
		t.Fatal("voter A VP should not be zero")
	}

	ratio := vpB.Quo(vpA)

	// Allow 1% tolerance from exact 10.0
	expectedRatio := math.LegacyNewDec(10)
	diff := ratio.Sub(expectedRatio).Abs()
	tolerance := math.LegacyMustNewDecFromStr("0.1") // 1% of 10

	if diff.GT(tolerance) {
		t.Fatalf("quadratic dampening: expected ratio ~10x, got %s (diff %s)", ratio, diff)
	}

	// Also verify B does NOT have 100x the VP of A
	hundredX := math.LegacyNewDec(100)
	if ratio.GT(hundredX.Sub(math.LegacyOneDec())) {
		t.Fatalf("whale should not have 100x VP: ratio = %s", ratio)
	}
}

// ---------------------------------------------------------------------------
// Test 3: Reputation multiplier effect -- higher rep gives higher VP
// ---------------------------------------------------------------------------

func TestQDRW_ReputationMultiplierEffect(t *testing.T) {
	handler, k, ctx := setupQDRWTestHandler(t)

	// Enable QDRW
	cfg := k.GetConfig(ctx)
	cfg.QDRWConfig.Enabled = true
	if err := k.SetConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	voter := sdk.AccAddress([]byte("voter1______________"))
	staked := uint64(1_000_000)

	vpLowRep, err := handler.CalculateVotingPower(ctx, voter, staked, 0.1)
	if err != nil {
		t.Fatalf("unexpected error for low rep: %v", err)
	}

	vpHighRep, err := handler.CalculateVotingPower(ctx, voter, staked, 0.9)
	if err != nil {
		t.Fatalf("unexpected error for high rep: %v", err)
	}

	if !vpHighRep.GT(vpLowRep) {
		t.Fatalf("high reputation should yield higher VP: low rep VP=%s, high rep VP=%s",
			vpLowRep, vpHighRep)
	}
}

// ---------------------------------------------------------------------------
// Test 4: xQORE contribution -- with NilTokenomicsKeeper, VP = sqrt(staked) * repMultiplier
// ---------------------------------------------------------------------------

func TestQDRW_XQOREContribution(t *testing.T) {
	handler, k, ctx := setupQDRWTestHandler(t)

	// Enable QDRW
	cfg := k.GetConfig(ctx)
	cfg.QDRWConfig.Enabled = true
	if err := k.SetConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	voter := sdk.AccAddress([]byte("voter1______________"))
	staked := uint64(1_000_000)
	rep := 0.5

	vp, err := handler.CalculateVotingPower(ctx, voter, staked, rep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Since NilTokenomicsKeeper returns 0 xQORE:
	// VP = sqrt(1_000_000) * ReputationMultiplier(0.5)
	// sqrt(1_000_000) = 1000
	// ReputationMultiplier(0.5) uses sigmoid(6*(0.5-0.5)) = sigmoid(0) = 0.5
	// Result: 0.5 + 1.5 * 0.5 = 1.25
	// VP = 1000 * 1.25 = 1250

	expectedSqrt := math.LegacyNewDec(1000)
	// ReputationMultiplier at 0.5 should be ~1.25 (sigmoid(0) = 0.5, so 0.5 + 1.5*0.5 = 1.25)
	expectedRepMul := math.LegacyMustNewDecFromStr("1.25")
	expectedVP := expectedSqrt.Mul(expectedRepMul)

	diff := vp.Sub(expectedVP).Abs()
	tolerance := math.LegacyMustNewDecFromStr("0.01")
	if diff.GT(tolerance) {
		t.Fatalf("xQORE stub: expected VP ~%s, got %s (diff %s)", expectedVP, vp, diff)
	}
}

// ---------------------------------------------------------------------------
// Test 5: Edge case -- zero stake returns zero VP when QDRW is enabled
// ---------------------------------------------------------------------------

func TestQDRW_ZeroStakeReturnsZero(t *testing.T) {
	handler, k, ctx := setupQDRWTestHandler(t)

	// Enable QDRW
	cfg := k.GetConfig(ctx)
	cfg.QDRWConfig.Enabled = true
	if err := k.SetConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	voter := sdk.AccAddress([]byte("voter1______________"))

	vp, err := handler.CalculateVotingPower(ctx, voter, 0, 0.9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !vp.IsZero() {
		t.Fatalf("zero stake should yield zero VP, got %s", vp)
	}
}
