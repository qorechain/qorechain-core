package gasabstraction

import (
	"bytes"
	"context"
	"errors"
	"testing"

	protov2 "google.golang.org/protobuf/proto"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// ---- mocks ----

type feeTx struct {
	fee     sdk.Coins
	gas     uint64
	payer   sdk.AccAddress
	granter sdk.AccAddress
}

func (t feeTx) GetMsgs() []sdk.Msg                    { return nil }
func (t feeTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }
func (t feeTx) GetGas() uint64                        { return t.gas }
func (t feeTx) GetFee() sdk.Coins                     { return t.fee }
func (t feeTx) FeePayer() []byte                      { return t.payer }
func (t feeTx) FeeGranter() []byte                    { return t.granter }

type memKeeper struct {
	mockGasAbstractionKeeper
	used   map[int64]math.Int
	config *types.GasAbstractionConfig
}

func (k *memKeeper) GetConfig(sdk.Context) types.GasAbstractionConfig { return *k.config }
func (k *memKeeper) BlockConversionUsed(ctx sdk.Context) math.Int {
	if v, ok := k.used[ctx.BlockHeight()]; ok {
		return v
	}
	return math.ZeroInt()
}
func (k *memKeeper) AddBlockConversionUsed(ctx sdk.Context, a math.Int) {
	k.used[ctx.BlockHeight()] = k.BlockConversionUsed(ctx).Add(a)
}

type memBank struct{ bal map[string]sdk.Coins }

func (b *memBank) GetBalance(_ context.Context, a sdk.AccAddress, d string) sdk.Coin {
	return sdk.NewCoin(d, b.bal[a.String()].AmountOf(d))
}
func (b *memBank) GetAllBalances(_ context.Context, a sdk.AccAddress) sdk.Coins {
	return b.bal[a.String()]
}
func (b *memBank) move(from, to sdk.AccAddress, amt sdk.Coins) error {
	have := b.bal[from.String()]
	if !have.IsAllGTE(amt) {
		return errors.New("insufficient funds")
	}
	b.bal[from.String()] = have.Sub(amt...)
	b.bal[to.String()] = b.bal[to.String()].Add(amt...)
	return nil
}
func (b *memBank) SendCoinsFromAccountToModule(_ context.Context, from sdk.AccAddress, mod string, amt sdk.Coins) error {
	return b.move(from, authtypes.NewModuleAddress(mod), amt)
}
func (b *memBank) SendCoinsFromModuleToModule(_ context.Context, from, to string, amt sdk.Coins) error {
	return b.move(authtypes.NewModuleAddress(from), authtypes.NewModuleAddress(to), amt)
}
func (b *memBank) SendCoinsFromModuleToAccount(_ context.Context, from string, to sdk.AccAddress, amt sdk.Coins) error {
	return b.move(authtypes.NewModuleAddress(from), to, amt)
}

type memAccounts map[string]bool

func (m memAccounts) GetAccount(_ context.Context, a sdk.AccAddress) sdk.AccountI {
	if m[a.String()] {
		return authtypes.NewBaseAccountWithAddress(a)
	}
	return nil
}

type memGrants struct{ used sdk.Coins }

func (g *memGrants) UseGrantedFees(_ context.Context, _, _ sdk.AccAddress, fee sdk.Coins, _ []sdk.Msg) error {
	g.used = fee
	return nil
}

type memPools map[uint64]ammtypes.Pool

func (p memPools) GetPool(_ sdk.Context, id uint64) (ammtypes.Pool, bool) {
	pool, ok := p[id]
	return pool, ok
}

// recorder is a decorator that records that it ran and what fee it saw.
type recorder struct {
	name string
	log  *[]string
	fee  *sdk.Coins
	err  error
}

func (r recorder) AnteHandle(ctx sdk.Context, tx sdk.Tx, sim bool, next sdk.AnteHandler) (sdk.Context, error) {
	*r.log = append(*r.log, r.name)
	if r.fee != nil {
		*r.fee = tx.(sdk.FeeTx).GetFee()
	}
	if r.err != nil {
		return ctx, r.err
	}
	return next(ctx, tx, sim)
}

// ---- fixture ----

var (
	payer   = sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	granter = sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	float   = authtypes.NewModuleAddress(types.ModuleName)
	feeColl = authtypes.NewModuleAddress(authtypes.FeeCollectorName)
)

const usdc = "ibc/0000000000000000000000000000000000000000000000000000000000000001"

type fixture struct {
	on       bool
	cfg      types.GasAbstractionConfig
	keeper   *memKeeper
	bank     *memBank
	grants   *memGrants
	pools    memPools
	log      []string
	seenFee  sdk.Coins
	checker  func(sdk.Context, sdk.Tx) (sdk.Coins, int64, error)
	minPerGs math.LegacyDec
}

func newFixture() *fixture {
	f := &fixture{on: true, minPerGs: math.LegacyNewDec(10)}
	f.cfg = types.DefaultGasAbstractionConfig()
	f.cfg.AcceptedTokens = append(f.cfg.AcceptedTokens, types.AcceptedFeeToken{
		Denom: usdc, ConversionRate: "20", PriceSource: types.PriceSourceStatic, RateExpiryHeight: 1_000,
	})
	f.cfg.BlockConversionCap = "10000000"
	f.cfg.FloatMin = "1000"
	f.keeper = &memKeeper{used: map[int64]math.Int{}, config: &f.cfg}
	f.bank = &memBank{bal: map[string]sdk.Coins{
		payer.String():   sdk.NewCoins(sdk.NewInt64Coin(usdc, 1_000_000)),
		granter.String(): sdk.NewCoins(sdk.NewInt64Coin(usdc, 1_000_000)),
		float.String():   sdk.NewCoins(sdk.NewInt64Coin("uqor", 5_000_000)),
	}}
	f.grants = &memGrants{}
	f.pools = memPools{}
	// Default checker: charge the whole converted value, priority = value.
	f.checker = func(_ sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		fee := tx.(sdk.FeeTx).GetFee()
		return fee, fee.AmountOf("uqor").Int64(), nil
	}
	return f
}

func (f *fixture) router() FeeRouterDecorator {
	return NewFeeRouterDecorator(FeeRouterOptions{
		Enabled: func(sdk.Context) bool { return f.on },
		Native: []sdk.AnteDecorator{
			recorder{name: "native-mgp", log: &f.log},
			recorder{name: "native-size", log: &f.log},
			recorder{name: "native-deduct", log: &f.log},
		},
		Keeper:         f.keeper,
		BankKeeper:     f.bank,
		AccountKeeper:  memAccounts{payer.String(): true, granter.String(): true},
		FeegrantKeeper: f.grants,
		Pools:          f.pools,
		TxSizeGas:      recorder{name: "conv-size", log: &f.log},
		MinGasPrice:    recorder{name: "conv-mgp", log: &f.log, fee: &f.seenFee},
		FeeChecker:     func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) { return f.checker(ctx, tx) },
		MinFeePerGas:   func(sdk.Context) math.LegacyDec { return f.minPerGs },
	})
}

func ctxAt(h int64) sdk.Context {
	return sdk.NewContext(nil, cmtproto.Header{Height: h}, false, log.NewNopLogger())
}

var final = func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }

func (f *fixture) run(tx feeTx, sim bool) (sdk.Context, error) {
	return f.router().AnteHandle(ctxAt(500).WithEventManager(sdk.NewEventManager()), tx, sim, final)
}

func usdcTx(amount int64) feeTx {
	return feeTx{fee: sdk.NewCoins(sdk.NewInt64Coin(usdc, amount)), gas: 100_000, payer: payer}
}

// ---- tests ----

func TestRouterOffTakesTheNativePathForEveryFee(t *testing.T) {
	f := newFixture()
	f.on = false
	for _, tx := range []feeTx{usdcTx(100), {fee: sdk.NewCoins(sdk.NewInt64Coin("uqor", 5)), gas: 1, payer: payer}} {
		f.log = nil
		if _, err := f.run(tx, false); err != nil {
			t.Fatal(err)
		}
		if got := f.log; len(got) != 3 || got[0] != "native-mgp" || got[2] != "native-deduct" {
			t.Fatalf("path %v", got)
		}
	}
	if !f.bank.bal[float.String()].AmountOf("uqor").Equal(math.NewInt(5_000_000)) {
		t.Fatal("the float moved with the switch off")
	}
}

func TestRouterOnNativeAndMultiDenomFeesTakeTheNativePath(t *testing.T) {
	f := newFixture()
	for _, fee := range []sdk.Coins{
		sdk.NewCoins(sdk.NewInt64Coin("uqor", 5)),
		sdk.NewCoins(sdk.NewInt64Coin("uqor", 5), sdk.NewInt64Coin(usdc, 5)),
		nil,
	} {
		f.log = nil
		if _, err := f.run(feeTx{fee: fee, gas: 1, payer: payer}, false); err != nil {
			t.Fatal(err)
		}
		if len(f.log) != 3 || f.log[0] != "native-mgp" {
			t.Fatalf("fee %s took %v", fee, f.log)
		}
	}
}

func TestConversionStaticRate(t *testing.T) {
	f := newFixture()
	// 1,000 units x rate 20 x (1 - 3%) = 19,400 uqor.
	ctx, err := f.run(usdcTx(1_000), false)
	if err != nil {
		t.Fatal(err)
	}
	if f.log[0] != "conv-size" || f.log[1] != "conv-mgp" || len(f.log) != 2 {
		t.Fatalf("path %v", f.log)
	}
	if !f.seenFee.Equal(sdk.NewCoins(sdk.NewInt64Coin("uqor", 19_400))) {
		t.Fatalf("native checks saw %s, want 19400uqor", f.seenFee)
	}
	if got := f.bank.bal[payer.String()].AmountOf(usdc); !got.Equal(math.NewInt(999_000)) {
		t.Fatalf("payer left with %s", got)
	}
	if got := f.bank.bal[float.String()].AmountOf(usdc); !got.Equal(math.NewInt(1_000)) {
		t.Fatalf("float got %s foreign", got)
	}
	if got := f.bank.bal[feeColl.String()]; !got.Equal(sdk.NewCoins(sdk.NewInt64Coin("uqor", 19_400))) {
		t.Fatalf("fee collector got %s; it must only ever receive the native denom", got)
	}
	if !f.keeper.BlockConversionUsed(ctxAt(500)).Equal(math.NewInt(19_400)) {
		t.Fatal("block usage not recorded")
	}
	if ctx.Priority() != 19_400 {
		t.Fatalf("priority %d", ctx.Priority())
	}
	var sawTx, sawConv bool
	for _, ev := range ctx.EventManager().Events() {
		sawTx = sawTx || ev.Type == sdk.EventTypeTx
		sawConv = sawConv || ev.Type == types.EventTypeFeeConverted
	}
	if !sawTx || !sawConv {
		t.Fatal("fee events missing")
	}
}

func TestConversionRefusals(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*fixture) feeTx
		want error
	}{
		{"unaccepted denom", func(f *fixture) feeTx {
			return feeTx{fee: sdk.NewCoins(sdk.NewInt64Coin("ibc/FFFF", 10)), gas: 1, payer: payer}
		}, types.ErrUnsupportedDenom},
		{"listed without a source", func(f *fixture) feeTx {
			f.cfg.AcceptedTokens[1].PriceSource = ""
			return usdcTx(1000)
		}, types.ErrUnsupportedDenom},
		{"expired static rate", func(f *fixture) feeTx {
			f.cfg.AcceptedTokens[1].RateExpiryHeight = 499
			return usdcTx(1000)
		}, types.ErrConversionFailed},
		{"conversion closed", func(f *fixture) feeTx {
			f.cfg.BlockConversionCap = "0"
			return usdcTx(1000)
		}, types.ErrConversionFailed},
		{"block cap reached", func(f *fixture) feeTx {
			f.cfg.BlockConversionCap = "19399"
			return usdcTx(1000)
		}, types.ErrConversionFailed},
		{"float at its minimum", func(f *fixture) feeTx {
			f.cfg.FloatMin = "4990000"
			return usdcTx(1000)
		}, types.ErrConversionFailed},
		{"above the tip ceiling", func(f *fixture) feeTx {
			f.minPerGs = math.LegacyNewDecWithPrec(1, 2) // 0.01/gas x 100k gas x 3 = 3,000 < 19,400
			return usdcTx(1000)
		}, types.ErrConversionFailed},
		{"disabled module", func(f *fixture) feeTx {
			f.cfg.Enabled = false
			return usdcTx(1000)
		}, types.ErrGasAbstractionOff},
		{"worth nothing", func(f *fixture) feeTx { return usdcTx(0 + 0) }, types.ErrInsufficientFee},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture()
			tx := tc.mut(f)
			if tc.name == "worth nothing" {
				f.cfg.AcceptedTokens[1].ConversionRate = "0.00000001"
				tx = usdcTx(1)
			}
			before := f.bank.bal[payer.String()]
			for _, sim := range []bool{true, false} {
				_, err := f.run(tx, sim)
				if !errors.Is(err, tc.want) {
					t.Fatalf("simulate=%v: got %v, want %v", sim, err, tc.want)
				}
			}
			if !f.bank.bal[payer.String()].Equal(before) {
				t.Fatal("a refused conversion moved funds")
			}
		})
	}
}

func TestConversionFeeGranterPays(t *testing.T) {
	f := newFixture()
	tx := usdcTx(1000)
	tx.granter = granter
	if _, err := f.run(tx, false); err != nil {
		t.Fatal(err)
	}
	if !f.grants.used.Equal(sdk.NewCoins(sdk.NewInt64Coin(usdc, 1000))) {
		t.Fatalf("grant charged %s", f.grants.used)
	}
	if !f.bank.bal[payer.String()].AmountOf(usdc).Equal(math.NewInt(1_000_000)) ||
		!f.bank.bal[granter.String()].AmountOf(usdc).Equal(math.NewInt(999_000)) {
		t.Fatal("the granter did not pay")
	}
}

func TestConversionChargesTheShareTheDynamicFeeTakes(t *testing.T) {
	f := newFixture()
	// The dynamic-fee checker charges 9,700 of the 19,400 value (half).
	f.checker = func(_ sdk.Context, _ sdk.Tx) (sdk.Coins, int64, error) {
		return sdk.NewCoins(sdk.NewInt64Coin("uqor", 9_700)), 7, nil
	}
	if _, err := f.run(usdcTx(1_000), false); err != nil {
		t.Fatal(err)
	}
	if got := f.bank.bal[payer.String()].AmountOf(usdc); !got.Equal(math.NewInt(999_500)) {
		t.Fatalf("payer charged %s, want half (500)", math.NewInt(1_000_000).Sub(got))
	}
	if !f.bank.bal[feeColl.String()].AmountOf("uqor").Equal(math.NewInt(9_700)) {
		t.Fatal("fee collector amount wrong")
	}
}

func TestConversionUnknownPayer(t *testing.T) {
	f := newFixture()
	tx := usdcTx(1000)
	tx.payer = sdk.AccAddress(bytes.Repeat([]byte{9}, 20))
	if _, err := f.run(tx, false); err == nil {
		t.Fatal("a payer without an account was charged")
	}
}

func TestResolveRateAMM(t *testing.T) {
	cfg := types.DefaultGasAbstractionConfig()
	tok := types.AcceptedFeeToken{Denom: usdc, PriceSource: types.PriceSourceAMMTWAP, PoolID: 7, MinLiquidity: "1000000", MaxDeviationBps: 200}
	pool := ammtypes.Pool{
		ID: 7, Status: ammtypes.PoolStatusActive, TokenA: usdc, TokenB: "uqor",
		ReserveA: math.NewInt(100_000), ReserveB: math.NewInt(2_000_000), // spot 20 uqor per unit
		WeightedAvgPrice: math.LegacyNewDec(20),
	}
	ctx := ctxAt(10)
	rate, err := types.ResolveRate(ctx, cfg, tok, memPools{7: pool})
	if err != nil || !rate.Equal(math.LegacyNewDec(20)) {
		t.Fatalf("rate %s, %v", rate, err)
	}

	// Foreign denom as TokenB: the price is inverted.
	inv := pool
	inv.TokenA, inv.TokenB = "aaa", usdc
	inv.TokenA = "uqor"
	inv.ReserveA, inv.ReserveB = math.NewInt(2_000_000), math.NewInt(100_000)
	inv.WeightedAvgPrice = math.LegacyNewDecWithPrec(5, 2) // 0.05 unit per uqor
	rate, err = types.ResolveRate(ctx, cfg, tok, memPools{7: inv})
	if err != nil || !rate.Equal(math.LegacyNewDec(20)) {
		t.Fatalf("inverted rate %s, %v", rate, err)
	}

	thin := pool
	thin.ReserveB = math.NewInt(999_999)
	thin.ReserveA = math.NewInt(50_000)
	if _, err := types.ResolveRate(ctx, cfg, tok, memPools{7: thin}); !errors.Is(err, types.ErrConversionFailed) {
		t.Fatalf("thin pool: %v", err)
	}

	moved := pool
	moved.ReserveB = math.NewInt(2_100_000) // spot 21: 4.8% off the average
	if _, err := types.ResolveRate(ctx, cfg, tok, memPools{7: moved}); !errors.Is(err, types.ErrConversionFailed) {
		t.Fatalf("manipulated spot: %v", err)
	}

	paused := pool
	paused.Status = ammtypes.PoolStatusPaused
	if _, err := types.ResolveRate(ctx, cfg, tok, memPools{7: paused}); !errors.Is(err, types.ErrConversionFailed) {
		t.Fatalf("paused pool: %v", err)
	}
	wrongPair := pool
	wrongPair.TokenB = "ibc/OTHER"
	if _, err := types.ResolveRate(ctx, cfg, tok, memPools{7: wrongPair}); !errors.Is(err, types.ErrConversionFailed) {
		t.Fatalf("wrong pair: %v", err)
	}
	if _, err := types.ResolveRate(ctx, cfg, tok, memPools{}); !errors.Is(err, types.ErrConversionFailed) {
		t.Fatalf("missing pool: %v", err)
	}
}

// The old decorator now tells a simulation with a non-native fee what
// delivery would say; an empty fee (the usual estimation) still passes.
func TestLegacyDecoratorRefusesInSimulation(t *testing.T) {
	cfg := types.DefaultGasAbstractionConfig()
	cfg.AcceptedTokens = append(cfg.AcceptedTokens, types.AcceptedFeeToken{Denom: usdc, ConversionRate: "1"})
	d := NewGasAbstractionDecorator(mockGasAbstractionKeeper{enabled: true, config: cfg})
	ctx := ctxAt(5)
	if _, err := d.AnteHandle(ctx, feeTx{gas: 1, payer: payer}, true, final); err != nil {
		t.Fatalf("empty-fee simulation refused: %v", err)
	}
	if _, err := d.AnteHandle(ctx, feeTx{fee: sdk.NewCoins(sdk.NewInt64Coin("uqor", 1)), gas: 1, payer: payer}, true, final); err != nil {
		t.Fatalf("native-fee simulation refused: %v", err)
	}
	for _, denom := range []string{usdc, "ibc/NOPE"} {
		_, err := d.AnteHandle(ctx, feeTx{fee: sdk.NewCoins(sdk.NewInt64Coin(denom, 1)), gas: 1, payer: payer}, true, final)
		if !errors.Is(err, types.ErrUnsupportedDenom) {
			t.Fatalf("%s simulation: %v", denom, err)
		}
	}
}
