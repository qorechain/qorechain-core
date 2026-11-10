package amm

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/amm/types"
	"github.com/qorechain/qorechain-core/x/govauth"
)

// paramsKeeper records SetParams and validates like the real keeper.
type paramsKeeper struct {
	AMMKeeper
	set *types.Params
}

func (k *paramsKeeper) SetParams(_ sdk.Context, p types.Params) error {
	if err := p.Validate(); err != nil {
		return err
	}
	k.set = &p
	return nil
}

func updateParamsGate(t *testing.T, on bool) {
	t.Helper()
	prev := UpdateParamsEnabled
	UpdateParamsEnabled = func(sdk.Context) bool { return on }
	t.Cleanup(func() { UpdateParamsEnabled = prev })
}

func testCtx() sdk.Context {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	return testutil.DefaultContext(key, storetypes.NewTransientStoreKey("t_amm"))
}

func wantParams() types.Params {
	p := types.DefaultParams()
	p.SwapFeeBps = 25
	p.ProtocolFeeBps = 5
	p.PoolCreationFee = math.NewInt(50_000_000)
	return p
}

func TestUpdateParamsFromGovernance(t *testing.T) {
	updateParamsGate(t, true)
	k := &paramsKeeper{}
	ctx := testCtx()
	_, err := NewMsgServer(k).UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: govauth.Address(), Params: types.ParamsToView(wantParams()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if k.set == nil || k.set.SwapFeeBps != 25 || !k.set.PoolCreationFee.Equal(math.NewInt(50_000_000)) {
		t.Fatalf("params not stored: %+v", k.set)
	}
	if len(ctx.EventManager().Events()) != 1 {
		t.Fatalf("want one event, got %d", len(ctx.EventManager().Events()))
	}
}

func TestUpdateParamsRefusals(t *testing.T) {
	updateParamsGate(t, true)
	ctx := testCtx()
	bad := types.ParamsToView(wantParams())
	bad.ProtocolFeeBps = 40 // above swap_fee_bps
	cases := map[string]*types.MsgUpdateParams{
		"not governance":     {Authority: sdk.AccAddress(make([]byte, 20)).String(), Params: types.ParamsToView(wantParams())},
		"invalid params":     {Authority: govauth.Address(), Params: bad},
		"unparseable amount": {Authority: govauth.Address(), Params: types.ParamsView{MinLiquidity: "x", PoolCreationFee: "1", LpTokenDecimals: 6}},
	}
	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			k := &paramsKeeper{}
			if _, err := NewMsgServer(k).UpdateParams(ctx, msg); err == nil {
				t.Fatal("accepted")
			}
			if k.set != nil {
				t.Fatal("stored")
			}
		})
	}
}

// Below the amm-gov-params switch the message is refused even from
// governance, as if it did not exist.
func TestUpdateParamsBelowSwitch(t *testing.T) {
	updateParamsGate(t, false)
	k := &paramsKeeper{}
	if _, err := NewMsgServer(k).UpdateParams(testCtx(), &types.MsgUpdateParams{
		Authority: govauth.Address(), Params: types.ParamsToView(wantParams()),
	}); err == nil {
		t.Fatal("accepted below the switch")
	}
	UpdateParamsEnabled = nil
	if _, err := NewMsgServer(k).UpdateParams(testCtx(), &types.MsgUpdateParams{
		Authority: govauth.Address(), Params: types.ParamsToView(wantParams()),
	}); err == nil {
		t.Fatal("accepted with the switch unwired")
	}
	if k.set != nil {
		t.Fatal("stored")
	}
}

func TestParamsViewRoundTrip(t *testing.T) {
	p := wantParams()
	got, err := types.ParamsFromView(types.ParamsToView(p))
	if err != nil {
		t.Fatal(err)
	}
	if got.SwapFeeBps != p.SwapFeeBps || !got.MinLiquidity.Equal(p.MinLiquidity) || got.Enabled != p.Enabled ||
		got.LPTokenDecimals != p.LPTokenDecimals || got.MaxSwapImpactBps != p.MaxSwapImpactBps {
		t.Fatalf("round trip changed params: %+v vs %+v", got, p)
	}
}
