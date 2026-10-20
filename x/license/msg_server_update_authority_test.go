package license

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// authorityRecorder satisfies LicenseKeeper through the embedded interface and
// records the one call this message makes.
type authorityRecorder struct {
	LicenseKeeper
	set string
}

func (r *authorityRecorder) SetAuthority(_ sdk.Context, a string) { r.set = a }

func TestUpdateAuthority(t *testing.T) {
	ms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 10}, false, log.NewNopLogger())

	gov := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	other := authtypes.NewModuleAddress("someone").String()
	newAuth := authtypes.NewModuleAddress("new-authority").String()

	enabled := false
	prev := AuthorityUpdateEnabled
	AuthorityUpdateEnabled = func(sdk.Context) bool { return enabled }
	defer func() { AuthorityUpdateEnabled = prev }()

	rec := &authorityRecorder{}
	srv := NewMsgServer(rec)

	// Below the plan height the message does nothing.
	if _, err := srv.UpdateAuthority(ctx, &types.MsgUpdateAuthority{Authority: gov, NewAuthority: newAuth}); err == nil {
		t.Fatal("accepted before the feature was enabled")
	}
	enabled = true

	if _, err := srv.UpdateAuthority(ctx, &types.MsgUpdateAuthority{Authority: other, NewAuthority: newAuth}); err == nil {
		t.Fatal("accepted from an account other than governance")
	}
	if _, err := srv.UpdateAuthority(ctx, &types.MsgUpdateAuthority{Authority: gov, NewAuthority: "not-an-address"}); err == nil {
		t.Fatal("accepted an invalid new authority")
	}
	if rec.set != "" {
		t.Fatalf("authority changed by a refused message: %q", rec.set)
	}

	if _, err := srv.UpdateAuthority(ctx, &types.MsgUpdateAuthority{Authority: gov, NewAuthority: newAuth}); err != nil {
		t.Fatalf("governance update refused: %v", err)
	}
	if rec.set != newAuth {
		t.Fatalf("authority = %q, want %q", rec.set, newAuth)
	}
}
