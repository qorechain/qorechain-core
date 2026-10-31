package app

import (
	"errors"
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	rdkmod "github.com/qorechain/qorechain-core/x/rdk"
)

type rdkWithCursorSeed struct {
	rdkmod.RDKKeeper
	calls int
	err   error
}

func (f *rdkWithCursorSeed) SeedFinalizeCursors(sdk.Context) (int, error) {
	f.calls++
	return 0, f.err
}

type rdkWithoutCursorSeed struct{ rdkmod.RDKKeeper }

// The release step reaches the cursor seed only through the optional
// interface: a keeper without it (the public build's stub) makes the step a
// no-op, and a seed error fails the upgrade.
func TestUpgradeV3_2_5RDKSeedsTheCursors(t *testing.T) {
	ctx := sdk.Context{}.WithLogger(log.NewNopLogger())

	app := &QoreChainApp{RDKKeeper: rdkWithoutCursorSeed{}}
	if err := app.upgradeV3_2_5RDK(ctx); err != nil {
		t.Fatalf("keeper without the seed: %v", err)
	}

	k := &rdkWithCursorSeed{}
	app.RDKKeeper = k
	if err := app.upgradeV3_2_5RDK(ctx); err != nil || k.calls != 1 {
		t.Fatalf("seed calls=%d err=%v, want one call", k.calls, err)
	}

	k.err = errors.New("boom")
	if err := app.upgradeV3_2_5RDK(ctx); err == nil {
		t.Fatal("a seed error must fail the upgrade")
	}
}
