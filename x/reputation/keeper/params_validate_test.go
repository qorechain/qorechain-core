package keeper

import (
	"testing"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// QSR-2026-0601 item 2: SetParams refuses a set genesis validation would
// refuse, and writes nothing when it does.
func TestSetParamsValidates(t *testing.T) {
	k, ctx, key := setupRepV2(t, false)

	if err := k.SetParams(ctx, types.DefaultReputationParams()); err != nil {
		t.Fatalf("default params refused: %v", err)
	}
	before := dumpStore(ctx, key)

	zeroLambda := types.DefaultReputationParams()
	zeroLambda.Lambda = "0"
	if err := k.SetParams(ctx, zeroLambda); err == nil {
		t.Fatal("a zero lambda was accepted")
	}
	badSum := types.DefaultReputationParams()
	badSum.Alpha = "0.90"
	if err := k.SetParams(ctx, badSum); err == nil {
		t.Fatal("weights summing to 1.6 were accepted")
	}
	if after := dumpStore(ctx, key); after != before {
		t.Fatal("a refused SetParams changed the store")
	}

	// Genesis validation and SetParams apply one rule.
	if err := (types.GenesisState{Params: zeroLambda}).Validate(); err == nil {
		t.Fatal("genesis validation accepted a zero lambda")
	}
}

// The v3.2.2 upgrade handler rewrites the stored params as they are; it must
// replay exactly as it ran, so its writer does not validate.
func TestSetParamsAsStoredKeepsHistoricalWrite(t *testing.T) {
	k, ctx, _ := setupRepV2(t, false)
	p := types.DefaultReputationParams()
	p.Lambda = "0"
	p.EpochBlocks = types.DefaultEpochBlocks
	if err := k.SetParamsAsStored(ctx, p); err != nil {
		t.Fatal(err)
	}
	if got := k.GetParams(ctx); got != p {
		t.Fatalf("stored %+v, want %+v", got, p)
	}
}
