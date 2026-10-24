package burn

import (
	"errors"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/burn/types"
)

// foreignFeeTx is a transaction whose fee is in a denom the fee checker does
// not take, as it is after conversion (QSR-2026-0624 F-1, QSR-2026-0633).
type foreignFeeTx struct{ sdk.FeeTx }

func (foreignFeeTx) GetFee() sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin("uusdc", 100)) }

type convertedFeeKeyT struct{}

func TestBurnFailedFeeConvertedFee(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	cms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	payout := math.NewInt(4_200)
	source := func(ctx sdk.Context) (math.Int, bool) {
		v, ok := ctx.Value(convertedFeeKeyT{}).(math.Int)
		return v, ok
	}
	// The native checker refuses the foreign fee, as the real one does.
	checker := func(sdk.Context, sdk.Tx) (sdk.Coins, int64, error) { return nil, 0, errors.New("insufficient fee") }
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }

	for _, tc := range []struct {
		name      string
		withOpt   bool
		switchOn  bool
		converted bool
		want      math.Int
	}{
		{"no option (v3.2.4)", false, false, true, math.ZeroInt()},
		{"option, switch off", true, false, true, math.ZeroInt()},
		{"option, switch on, converted", true, true, true, payout},
		{"option, switch on, not converted", true, true, false, math.ZeroInt()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.NewContext(cms.CacheMultiStore(), cmtproto.Header{Height: 5}, false, log.NewNopLogger()).
				WithExecMode(sdk.ExecModeFinalize)
			if tc.converted {
				ctx = ctx.WithValue(convertedFeeKeyT{}, payout)
			}
			k := &accumulatorKeeper{key: key, on: true}
			dec := NewBurnFailedFeeDecorator(k, checker)
			reads := 0
			if tc.withOpt {
				dec = dec.WithConvertedFee(source, func(sdk.Context) bool { reads++; return tc.switchOn })
			}
			out, err := dec.AnteHandle(ctx, foreignFeeTx{}, false, next)
			if err != nil {
				t.Fatal(err)
			}
			if got := k.get(out); !got.Equal(tc.want) {
				t.Fatalf("recorded %s, want %s", got, tc.want)
			}
			if !tc.converted && reads != 0 {
				t.Fatal("the switch is read for a transaction that was not converted")
			}
			if fee, ok := out.Value(failedFeeKey{}).(math.Int); tc.want.IsPositive() && (!ok || !fee.Equal(payout)) {
				t.Fatalf("the post half would release %v, want %s", fee, payout)
			}
		})
	}
}
