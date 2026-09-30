package app

import (
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	bridgetypes "github.com/qorechain/qorechain-core/x/bridge/types"
)

type memChainConfigs struct {
	m     map[string]bridgetypes.ChainConfig
	order []string
}

func (s *memChainConfigs) GetAllChainConfigs(sdk.Context) []bridgetypes.ChainConfig {
	out := make([]bridgetypes.ChainConfig, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.m[id])
	}
	return out
}

func (s *memChainConfigs) SetChainConfig(_ sdk.Context, cc bridgetypes.ChainConfig) error {
	s.m[cc.ChainID] = cc
	return nil
}

// QSR-2026-0016: the handler folds known statuses to lower case, turns anything
// else into "paused", leaves valid ones alone, and is idempotent.
func TestNormaliseBridgeStatuses(t *testing.T) {
	store := &memChainConfigs{m: map[string]bridgetypes.ChainConfig{}}
	for id, st := range map[string]string{
		"ethereum": "pending", "solana": "PAUSED", "ton": " Active ", "bsc": "opne", "sui": "active",
	} {
		store.m[id] = bridgetypes.ChainConfig{ChainID: id, Status: bridgetypes.BridgeStatus(st)}
		store.order = append(store.order, id)
	}
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager()).WithLogger(log.NewNopLogger())
	if err := normaliseBridgeStatuses(ctx, store); err != nil {
		t.Fatal(err)
	}
	want := map[string]bridgetypes.BridgeStatus{
		"ethereum": "pending", "solana": "paused", "ton": "active", "bsc": "paused", "sui": "active",
	}
	for id, st := range want {
		if store.m[id].Status != st {
			t.Errorf("%s: %q, want %q", id, store.m[id].Status, st)
		}
	}
	if n := len(ctx.EventManager().Events()); n != 3 {
		t.Fatalf("%d events, want 3 (one per changed route)", n)
	}
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	if err := normaliseBridgeStatuses(ctx, store); err != nil {
		t.Fatal(err)
	}
	if n := len(ctx.EventManager().Events()); n != 0 {
		t.Fatalf("second run changed %d routes", n)
	}
}
