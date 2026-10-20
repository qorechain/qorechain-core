package app

import (
	"errors"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/burn"
	burntypes "github.com/qorechain/qorechain-core/x/burn/types"
)

// plainBurnKeeper has no stranded reserve correction, like the public stub.
type plainBurnKeeper struct{ burn.BurnKeeper }

type correctingBurnKeeper struct {
	burn.BurnKeeper
	changed bool
	err     error
	calls   int
}

func (c *correctingBurnKeeper) CorrectStrandedReserve(sdk.Context) (burntypes.StrandedRewards, burntypes.StrandedRewards, bool, error) {
	c.calls++
	before := burntypes.StrandedRewards{Reserved: math.NewInt(6_693_045_921_330), Released: math.ZeroInt(), Initialized: true}
	after := before
	if c.changed {
		after.Reserved = math.NewInt(10_500_542_706_859)
	}
	return before, after, c.changed, c.err
}

// QSR-2026-0341: the v3.2.3 step calls the burn keeper's correction, reports
// a change with an event, propagates an error, and is a no-op on a keeper
// without the correction (the public stub).
func TestUpgradeV323EconCorrectsStrandedReserve(t *testing.T) {
	newCtx := func() sdk.Context {
		return sdk.Context{}.WithEventManager(sdk.NewEventManager()).WithLogger(log.NewNopLogger())
	}

	k := &correctingBurnKeeper{changed: true}
	app := &QoreChainApp{BurnKeeper: k}
	ctx := newCtx()
	if err := app.upgradeV3_2_3Econ(ctx); err != nil {
		t.Fatal(err)
	}
	if k.calls != 1 || countEvents(ctx, "burn_stranded_reserve_corrected") != 1 {
		t.Fatalf("calls %d, events %d", k.calls, countEvents(ctx, "burn_stranded_reserve_corrected"))
	}

	k = &correctingBurnKeeper{}
	app.BurnKeeper = k
	ctx = newCtx()
	if err := app.upgradeV3_2_3Econ(ctx); err != nil || countEvents(ctx, "burn_stranded_reserve_corrected") != 0 {
		t.Fatalf("unchanged reserve: err %v events %d", err, countEvents(ctx, "burn_stranded_reserve_corrected"))
	}

	app.BurnKeeper = &correctingBurnKeeper{err: errors.New("boom")}
	if err := app.upgradeV3_2_3Econ(newCtx()); err == nil {
		t.Fatal("error was swallowed")
	}

	app.BurnKeeper = plainBurnKeeper{}
	if err := app.upgradeV3_2_3Econ(newCtx()); err != nil {
		t.Fatal(err)
	}
}
