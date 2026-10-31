package gasabstraction

import (
	"errors"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// Fee pricing from the gasabstraction-pool-type-pricing switch (QSR-2026-0634).

// memQuoter is a pool reader that also quotes: every quote answers out (or
// err) and charges quoteGas on the context's meter, as the AMM's reads do.
type memQuoter struct {
	memPools
	out   math.Int
	err   error
	calls int
}

const quoteGas = 1_234

func (q *memQuoter) QuoteExactIn(ctx sdk.Context, _ uint64, _ string, _ math.Int) (math.Int, math.Int, error) {
	q.calls++
	ctx.GasMeter().ConsumeGas(quoteGas, "quote")
	if q.err != nil {
		return math.Int{}, math.Int{}, q.err
	}
	return q.out, math.ZeroInt(), nil
}

// reporterFixture is the reported run: a stable_swap pool skewed by a swap to
// 950,150,250 foreign / 1,049,950,000 uqor whose average has caught up with
// the skewed ratio (1.0504 uqor per unit), while the pool really pays 398,804
// uqor for the 400,000-unit fee. 3% premium, 2,000 bps deviation, 3x ceiling.
func reporterFixture() *fixture {
	f := newFixture()
	f.cfg.AcceptedTokens[1] = types.AcceptedFeeToken{
		Denom: usdc, ConversionRate: "1", PriceSource: types.PriceSourceAMMTWAP,
		PoolID: 1, MinLiquidity: "100000000", MaxDeviationBps: 2_000,
	}
	f.quoter = &memQuoter{memPools: memPools{1: {
		ID: 1, Type: ammtypes.PoolTypeStableSwap, Status: ammtypes.PoolStatusActive,
		TokenA: usdc, TokenB: "uqor",
		ReserveA: math.NewInt(950_150_250), ReserveB: math.NewInt(1_049_950_000),
		WeightedAvgPrice: math.LegacyMustNewDecFromStr("1.050381395915677606"),
	}}, out: math.NewInt(398_804)}
	f.minPerGs = math.LegacyNewDec(1)
	f.cfg.BlockConversionCap = "20000000"
	f.bank.bal[payer.String()] = sdk.NewCoins(sdk.NewInt64Coin(usdc, 1_000_000))
	return f
}

func reporterTx() feeTx {
	return feeTx{fee: sdk.NewCoins(sdk.NewInt64Coin(usdc, 400_000)), gas: 1_500_000, payer: payer}
}

// runGas runs tx and returns the gas the router consumed.
func (f *fixture) runGas(tx feeTx, sim bool) (uint64, error) {
	ctx := ctxAt(500).WithEventManager(sdk.NewEventManager()).WithGasMeter(storetypes.NewInfiniteGasMeter())
	_, err := f.router().AnteHandle(ctx, tx, sim, final)
	return ctx.GasMeter().GasConsumed(), err
}

func TestReporterStableSwapPricedFromTheAverageWithTheSwitchOff(t *testing.T) {
	f := reporterFixture()
	if _, err := f.run(reporterTx(), false); err != nil {
		t.Fatal(err)
	}
	// floor(400,000 x 1.050381395915677606 x 0.97) = 407,547, as reported.
	if got := f.bank.bal[feeColl.String()].AmountOf("uqor"); !got.Equal(math.NewInt(407_547)) {
		t.Fatalf("switch off paid %s, want the historical 407547", got)
	}
	if f.quoter.calls != 0 {
		t.Fatal("the switch-off path asked the pool for a quote")
	}
}

func TestReporterStableSwapRefusedWithTheSwitchOn(t *testing.T) {
	f := reporterFixture()
	f.pricing = true
	before := f.bank.bal[payer.String()]
	for _, sim := range []bool{true, false} {
		_, err := f.run(reporterTx(), sim)
		if !errors.Is(err, types.ErrConversionFailed) {
			t.Fatalf("simulate=%v: got %v, want ErrConversionFailed", sim, err)
		}
	}
	if !f.bank.bal[payer.String()].Equal(before) || f.bank.bal[feeColl.String()].AmountOf("uqor").IsPositive() {
		t.Fatal("a refused conversion moved funds")
	}
	if f.quoter.calls != 0 {
		t.Fatal("a refused pool type was quoted")
	}
}

// cpFixture is a constant_product pool at 20 uqor per unit, average 20.
func cpFixture(quote int64) *fixture {
	f := newFixture()
	f.cfg.AcceptedTokens[1] = types.AcceptedFeeToken{
		Denom: usdc, ConversionRate: "1", PriceSource: types.PriceSourceAMMTWAP,
		PoolID: 7, MinLiquidity: "1000000", MaxDeviationBps: 200,
	}
	f.quoter = &memQuoter{memPools: memPools{7: {
		ID: 7, Type: ammtypes.PoolTypeConstantProduct, Status: ammtypes.PoolStatusActive,
		TokenA: usdc, TokenB: "uqor",
		ReserveA: math.NewInt(100_000), ReserveB: math.NewInt(2_000_000),
		WeightedAvgPrice: math.LegacyNewDec(20),
	}}, out: math.NewInt(quote)}
	return f
}

func TestConstantProductValuedAtTheLowerOfAverageAndQuote(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pricing bool
		quote   int64
		want    int64
	}{
		// 1,000 units: average value 20,000; less 3% = 19,400.
		{"switch off ignores the quote", false, 15_000, 19_400},
		{"quote below the average", true, 19_000, 18_430}, // floor(19,000 x 0.97)
		{"quote above the average", true, 25_000, 19_400},
		{"quote equal to the average", true, 20_000, 19_400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := cpFixture(tc.quote)
			f.pricing = tc.pricing
			ctx, err := f.run(usdcTx(1_000), false)
			if err != nil {
				t.Fatal(err)
			}
			if got := f.bank.bal[feeColl.String()].AmountOf("uqor"); !got.Equal(math.NewInt(tc.want)) {
				t.Fatalf("paid %s, want %d", got, tc.want)
			}
			if !f.seenFee.Equal(sdk.NewCoins(sdk.NewInt64Coin("uqor", tc.want))) {
				t.Fatalf("native checks saw %s", f.seenFee)
			}
			if paid, ok := ConvertedFee(ctx); !ok || !paid.Equal(math.NewInt(tc.want)) {
				t.Fatalf("converted fee %v %v", paid, ok)
			}
			if wantCalls := map[bool]int{false: 0, true: 1}[tc.pricing]; f.quoter.calls != wantCalls {
				t.Fatalf("quoted %d times, want %d", f.quoter.calls, wantCalls)
			}
		})
	}
}

func TestPoolPricingRefusalsWithTheSwitchOn(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*fixture)
	}{
		{"pool refuses the quote", func(f *fixture) { f.quoter.err = ammtypes.ErrPriceImpactExceeded }},
		{"quote worth nothing", func(f *fixture) { f.quoter.out = math.ZeroInt() }},
		{"no quoter", func(f *fixture) { f.pools, f.quoter = f.quoter.memPools, nil }},
		{"unknown pool type", func(f *fixture) {
			p := f.quoter.memPools[7]
			p.Type = "weighted"
			f.quoter.memPools[7] = p
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := cpFixture(19_000)
			f.pricing = true
			tc.mut(f)
			before := f.bank.bal[payer.String()]
			for _, sim := range []bool{true, false} {
				if _, err := f.run(usdcTx(1_000), sim); err == nil {
					t.Fatalf("simulate=%v: converted", sim)
				}
			}
			if !f.bank.bal[payer.String()].Equal(before) {
				t.Fatal("a refused conversion moved funds")
			}
		})
	}
	// The same pools convert with the switch off, as before.
	f := cpFixture(0)
	f.quoter.err = ammtypes.ErrPriceImpactExceeded
	if _, err := f.run(usdcTx(1_000), false); err != nil {
		t.Fatalf("switch off: %v", err)
	}
}

// The quote is read in simulation, CheckTx and delivery alike, so an estimate
// covers what delivery charges.
func TestPoolPricingGasSameInSimulationAndDelivery(t *testing.T) {
	var gas [2]uint64
	for i, sim := range []bool{true, false} {
		f := cpFixture(19_000)
		f.pricing = true
		g, err := f.runGas(usdcTx(1_000), sim)
		if err != nil {
			t.Fatal(err)
		}
		gas[i] = g
	}
	if gas[0] != gas[1] || gas[0] < quoteGas {
		t.Fatalf("gas simulate %d, deliver %d", gas[0], gas[1])
	}
	// With the switch off nothing is quoted: the gas is what it was.
	f := cpFixture(19_000)
	g, err := f.runGas(usdcTx(1_000), false)
	if err != nil || g != gas[1]-quoteGas {
		t.Fatalf("switch off gas %d (%v), want %d", g, err, gas[1]-quoteGas)
	}
}

func TestPerTxBoundWithZeroMinimumFee(t *testing.T) {
	// Switch off: a zero minimum fee per gas skips the bound (historical).
	f := newFixture()
	f.minPerGs = math.LegacyZeroDec()
	if _, err := f.run(usdcTx(1_000), false); err != nil {
		t.Fatalf("switch off: %v", err)
	}

	// Switch on: refused, in simulation too, and nothing moves.
	f = newFixture()
	f.pricing = true
	f.minPerGs = math.LegacyZeroDec()
	before := f.bank.bal[payer.String()]
	for _, sim := range []bool{true, false} {
		if _, err := f.run(usdcTx(1_000), sim); !errors.Is(err, types.ErrConversionFailed) {
			t.Fatalf("simulate=%v: got %v, want ErrConversionFailed", sim, err)
		}
	}
	if !f.bank.bal[payer.String()].Equal(before) || f.bank.bal[feeColl.String()].AmountOf("uqor").IsPositive() {
		t.Fatal("a refused conversion moved funds")
	}

	// Switch on, zero gas: no bound can be computed, refused.
	f = newFixture()
	f.pricing = true
	tx := usdcTx(1_000)
	tx.gas = 0
	if _, err := f.run(tx, false); !errors.Is(err, types.ErrConversionFailed) {
		t.Fatalf("zero gas: %v", err)
	}

	// Switch on, a positive minimum: the bound applies as before.
	f = newFixture()
	f.pricing = true
	if _, err := f.run(usdcTx(1_000), false); err != nil {
		t.Fatalf("positive minimum: %v", err)
	}

	// max_tip_multiplier 0 is governance's documented "no bound": unchanged.
	f = newFixture()
	f.pricing = true
	f.minPerGs = math.LegacyZeroDec()
	f.cfg.MaxTipMultiplier = 0
	if _, err := f.run(usdcTx(1_000), false); err != nil {
		t.Fatalf("no bound configured: %v", err)
	}
}

func TestResolveFeeRatePoolTypes(t *testing.T) {
	cfg := types.DefaultGasAbstractionConfig()
	tok := types.AcceptedFeeToken{Denom: usdc, PriceSource: types.PriceSourceAMMTWAP, PoolID: 7, MinLiquidity: "1000000", MaxDeviationBps: 200}
	pool := ammtypes.Pool{
		ID: 7, Type: ammtypes.PoolTypeConstantProduct, Status: ammtypes.PoolStatusActive, TokenA: usdc, TokenB: "uqor",
		ReserveA: math.NewInt(100_000), ReserveB: math.NewInt(2_000_000), WeightedAvgPrice: math.LegacyNewDec(20),
	}
	ctx := ctxAt(10)
	if r, err := types.ResolveFeeRate(ctx, cfg, tok, memPools{7: pool}); err != nil || !r.Equal(math.LegacyNewDec(20)) {
		t.Fatalf("constant_product: %s %v", r, err)
	}
	for _, typ := range []ammtypes.PoolType{ammtypes.PoolTypeStableSwap, "", "weighted"} {
		p := pool
		p.Type = typ
		if _, err := types.ResolveFeeRate(ctx, cfg, tok, memPools{7: p}); !errors.Is(err, types.ErrConversionFailed) {
			t.Fatalf("type %q: %v", typ, err)
		}
		// ResolveRate (below the switch) does not look at the type.
		if r, err := types.ResolveRate(ctx, cfg, tok, memPools{7: p}); err != nil || !r.Equal(math.LegacyNewDec(20)) {
			t.Fatalf("type %q below the switch: %s %v", typ, r, err)
		}
	}
	// A static rate is not a pool price: unaffected.
	st := types.AcceptedFeeToken{Denom: usdc, ConversionRate: "3", PriceSource: types.PriceSourceStatic, RateExpiryHeight: 100}
	if r, err := types.ResolveFeeRate(ctx, cfg, st, nil); err != nil || !r.Equal(math.LegacyNewDec(3)) {
		t.Fatalf("static: %s %v", r, err)
	}
	if v, err := types.FeeValue(ctx, cfg, st, math.NewInt(1_000), math.LegacyNewDec(3), nil); err != nil || !v.Equal(types.NativeValue(math.NewInt(1_000), math.LegacyNewDec(3), cfg.PremiumBps)) {
		t.Fatalf("static value: %s %v", v, err)
	}
}
