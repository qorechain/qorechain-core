package cmd

import (
	"sync"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/qorechain/qorechain-core/app"
	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
)

var (
	testAppOnce sync.Once
	testApp     *app.QoreChainApp
)

// sharedTestApp builds the app once per test binary: the EVM chain config it
// sets is process global and refuses a second setting.
func sharedTestApp(t *testing.T) *app.QoreChainApp {
	t.Helper()
	testAppOnce.Do(func() {
		opts := simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}
		testApp = app.NewQoreChainApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, opts)
	})
	return testApp
}

// Reputation scoring v2 needs its gov hook (participation) and staking hook
// (removal) installed by depinject during Build, next to the existing ones.
func TestReputationHooksInstalled(t *testing.T) {
	a := sharedTestApp(t)

	staking, ok := a.StakingKeeper.Hooks().(stakingtypes.MultiStakingHooks)
	if !ok {
		t.Fatalf("staking hooks are %T", a.StakingKeeper.Hooks())
	}
	found := 0
	for _, h := range staking {
		if w, ok := h.(stakingtypes.StakingHooksWrapper); ok {
			h = w.StakingHooks
		}
		if _, ok := h.(reputationkeeper.Hooks); ok {
			found++
		}
	}
	if found != 1 || len(staking) < 3 {
		t.Fatalf("reputation staking hook installed %d times among %d hooks", found, len(staking))
	}

	gov, ok := a.GovKeeper.Hooks().(govtypes.MultiGovHooks)
	if !ok {
		t.Fatalf("gov hooks are %T", a.GovKeeper.Hooks())
	}
	found = 0
	for _, h := range gov {
		if w, ok := h.(govtypes.GovHooksWrapper); ok {
			h = w.GovHooks
		}
		if _, ok := h.(reputationkeeper.Hooks); ok {
			found++
		}
	}
	if found != 1 || len(gov) < 2 {
		t.Fatalf("reputation gov hook installed %d times among %d hooks", found, len(gov))
	}
}
