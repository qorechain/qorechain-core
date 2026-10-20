package keeper

import (
	"bytes"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
	"github.com/qorechain/qorechain-core/x/qca/types"
)

// QSR-2026-0405: an unreadable stored config still falls back to the default
// (deterministic on every node), but no longer silently: it is logged at
// Error level.
func TestGetConfigFallbackIsLoggedAsError(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	if err := ms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	logger := log.NewLogger(&buf, log.ColorOption(false))
	k := NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), key, reputationkeeper.Keeper{}, nil, logger)
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 1}, false, logger)

	ctx.KVStore(key).Set(types.ConfigKey, []byte("{not json"))
	got := k.GetConfig(ctx)
	if want := types.DefaultQCAConfig(); got.UseReputationWeighting != want.UseReputationWeighting || got.MinReputationScore != want.MinReputationScore {
		t.Fatalf("fallback %+v, want the default", got)
	}
	out := buf.String()
	if !strings.Contains(out, "ERR") || !strings.Contains(out, "failed to unmarshal qca config") {
		t.Fatalf("no Error-level log: %q", out)
	}
}
