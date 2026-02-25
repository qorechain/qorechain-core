//go:build proprietary

package keeper

import (
	"testing"

	"cosmossdk.io/math"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// setupSlashingTestKeeper reuses the same pattern as setupBondingTestKeeper.
// Both files share the proprietary build tag so the helper is accessible.
func setupSlashingTestKeeper(t *testing.T) (Keeper, sdk.Context) {
	return setupBondingTestKeeper(t)
}

// ctxWithHeight returns a new context with the given block height.
func ctxWithHeight(ctx sdk.Context, height int64) sdk.Context {
	return ctx.WithBlockHeader(cmtproto.Header{Height: height})
}

// ---------------------------------------------------------------------------
// Test 1: First offense (no history)
// effective_count = 0 -> escalation^0 = 1 -> penalty = base_rate * severity
// With default base_rate=0.01 and severity=1.0, penalty = 0.01 (1%)
// ---------------------------------------------------------------------------

func TestSlashing_FirstOffense(t *testing.T) {
	k, ctx := setupSlashingTestKeeper(t)
	ctx = ctxWithHeight(ctx, 1000)

	severity := math.LegacyOneDec()
	penalty, err := k.ComputeProgressivePenalty(ctx, "qorvaloper1abc", 1000, "downtime", severity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decStr("0.01")
	diff := penalty.Sub(expected).Abs()
	if diff.GT(decStr("0.0001")) {
		t.Fatalf("first offense: expected ~%s, got %s (diff %s)", expected, penalty, diff)
	}
}

// ---------------------------------------------------------------------------
// Test 2: Repeated offense (recent)
// First offense at height 100, second at height 200.
// blocks_since = 100, halflife = 100000, exponent = 100/100000 = 0.001
// decay = 0.5^0.001 = exp(-ln2*0.001) ~ 0.999307 ~ 1.0
// effective_count ~ 1.0
// escalation^1.0 = 1.5
// penalty = 0.01 * 1.5 * 1.0 = 0.015
// ---------------------------------------------------------------------------

func TestSlashing_RepeatedRecentOffense(t *testing.T) {
	k, ctx := setupSlashingTestKeeper(t)

	// First offense at height 100
	ctx1 := ctxWithHeight(ctx, 100)
	severity := math.LegacyOneDec()
	_, err := k.ComputeProgressivePenalty(ctx1, "qorvaloper1abc", 100, "downtime", severity)
	if err != nil {
		t.Fatalf("unexpected error on first offense: %v", err)
	}

	// Second offense at height 200
	ctx2 := ctxWithHeight(ctx, 200)
	penalty, err := k.ComputeProgressivePenalty(ctx2, "qorvaloper1abc", 200, "downtime", severity)
	if err != nil {
		t.Fatalf("unexpected error on second offense: %v", err)
	}

	// Expected: ~0.015 (base_rate * escalation^~1.0 * severity)
	expected := decStr("0.015")
	diff := penalty.Sub(expected).Abs()
	if diff.GT(decStr("0.001")) {
		t.Fatalf("repeated recent offense: expected ~%s, got %s (diff %s)", expected, penalty, diff)
	}

	// Penalty should be greater than base rate (0.01)
	baseRate := decStr("0.01")
	if !penalty.GT(baseRate) {
		t.Fatalf("repeated offense should escalate: penalty %s <= base_rate %s", penalty, baseRate)
	}
}

// ---------------------------------------------------------------------------
// Test 3: Very old offense (decayed)
// First offense at height 0, current height = 500000 (5 half-lives).
// decay = 0.5^5 = 0.03125
// effective_count ~ 0.03125 -> barely any escalation
// penalty ~ base_rate * severity ~ 0.01
// ---------------------------------------------------------------------------

func TestSlashing_OldDecayedOffense(t *testing.T) {
	k, ctx := setupSlashingTestKeeper(t)

	// First offense at height 0
	ctx0 := ctxWithHeight(ctx, 0)
	severity := math.LegacyOneDec()
	_, err := k.ComputeProgressivePenalty(ctx0, "qorvaloper1xyz", 0, "downtime", severity)
	if err != nil {
		t.Fatalf("unexpected error on first offense: %v", err)
	}

	// Second offense at height 500000 (5 half-lives later)
	ctx2 := ctxWithHeight(ctx, 500_000)
	penalty, err := k.ComputeProgressivePenalty(ctx2, "qorvaloper1xyz", 500_000, "downtime", severity)
	if err != nil {
		t.Fatalf("unexpected error on second offense: %v", err)
	}

	// With 5 half-lives of decay, effective_count ~ 0.03125
	// escalation = 1.5^0.03125 ~ exp(0.03125 * ln(1.5)) ~ exp(0.03125 * 0.4055) ~ exp(0.01267) ~ 1.01275
	// penalty ~ 0.01 * 1.01275 * 1.0 ~ 0.010128
	// Should be very close to the base rate
	baseRate := decStr("0.01")
	diff := penalty.Sub(baseRate).Abs()
	if diff.GT(decStr("0.001")) {
		t.Fatalf("old decayed offense: penalty %s should be near base_rate %s (diff %s)", penalty, baseRate, diff)
	}
}

// ---------------------------------------------------------------------------
// Test 4: Cap at 33%
// Many recent offenses should cap penalty at MaxPenalty (0.33)
// ---------------------------------------------------------------------------

func TestSlashing_CappedAtMax(t *testing.T) {
	k, ctx := setupSlashingTestKeeper(t)

	severity := math.LegacyOneDec()
	maxPenalty := decStr("0.33")

	// Create many rapid infractions to build up effective_count
	var penalty math.LegacyDec
	var err error
	for i := int64(1); i <= 30; i++ {
		ctxI := ctxWithHeight(ctx, i*10)
		penalty, err = k.ComputeProgressivePenalty(ctxI, "qorvaloper1bad", i*10, "double_sign", severity)
		if err != nil {
			t.Fatalf("unexpected error on offense %d: %v", i, err)
		}
	}

	// The last penalty should be capped at maxPenalty
	if penalty.GT(maxPenalty) {
		t.Fatalf("penalty %s exceeds max_penalty %s", penalty, maxPenalty)
	}

	// After enough infractions, penalty should have reached the cap
	diff := penalty.Sub(maxPenalty).Abs()
	if diff.GT(decStr("0.001")) {
		t.Fatalf("after many infractions, penalty %s should be capped at %s (diff %s)", penalty, maxPenalty, diff)
	}
}

// ---------------------------------------------------------------------------
// Test 5: Stats counter increments
// ---------------------------------------------------------------------------

func TestSlashing_StatsIncrement(t *testing.T) {
	k, ctx := setupSlashingTestKeeper(t)

	statsBefore := k.GetStats(ctx)
	if statsBefore.SlashingEvents != 0 {
		t.Fatalf("expected 0 slashing events initially, got %d", statsBefore.SlashingEvents)
	}

	severity := math.LegacyOneDec()
	ctx1 := ctxWithHeight(ctx, 100)
	_, err := k.ComputeProgressivePenalty(ctx1, "qorvaloper1abc", 100, "downtime", severity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statsAfter := k.GetStats(ctx)
	if statsAfter.SlashingEvents != 1 {
		t.Fatalf("expected 1 slashing event after call, got %d", statsAfter.SlashingEvents)
	}

	// Second call should increment again
	ctx2 := ctxWithHeight(ctx, 200)
	_, err = k.ComputeProgressivePenalty(ctx2, "qorvaloper1abc", 200, "downtime", severity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statsAfter2 := k.GetStats(ctx)
	if statsAfter2.SlashingEvents != 2 {
		t.Fatalf("expected 2 slashing events after second call, got %d", statsAfter2.SlashingEvents)
	}
}
