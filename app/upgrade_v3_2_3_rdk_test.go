package app

import (
	"errors"
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	rdkmod "github.com/qorechain/qorechain-core/x/rdk"
)

type rdkWithBackfill struct {
	rdkmod.RDKKeeper
	calls int
	err   error
}

func (f *rdkWithBackfill) BackfillEscrowLedger(sdk.Context) (int, error) {
	f.calls++
	return 0, f.err
}

type rdkWithoutBackfill struct{ rdkmod.RDKKeeper }

// The release step reaches the backfill only through the optional interface:
// a keeper without it (the public build's stub) makes the step a no-op, and a
// backfill error fails the upgrade.
func TestUpgradeV3_2_3RdkRunsTheBackfill(t *testing.T) {
	ctx := sdk.Context{}.WithLogger(log.NewNopLogger())

	app := &QoreChainApp{RDKKeeper: rdkWithoutBackfill{}}
	if err := app.upgradeV3_2_3Rdk(ctx); err != nil {
		t.Fatalf("keeper without backfill: %v", err)
	}

	k := &rdkWithBackfill{}
	app.RDKKeeper = k
	if err := app.upgradeV3_2_3Rdk(ctx); err != nil || k.calls != 1 {
		t.Fatalf("backfill calls=%d err=%v, want one call", k.calls, err)
	}

	k.err = errors.New("boom")
	if err := app.upgradeV3_2_3Rdk(ctx); err == nil {
		t.Fatal("a backfill error must fail the upgrade")
	}
}
