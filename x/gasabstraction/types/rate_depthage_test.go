package types

import (
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

type onePool struct{ p ammtypes.Pool }

func (o onePool) GetPool(_ sdk.Context, id uint64) (ammtypes.Pool, bool) {
	return o.p, id == o.p.ID
}

func depthAgeGate(t *testing.T, on bool) {
	t.Helper()
	prev := DepthAgeGateEnabled
	DepthAgeGateEnabled = func(sdk.Context) bool { return on }
	t.Cleanup(func() { DepthAgeGateEnabled = prev })
}

// thinForeignPool is the case the gate exists for: a deep native side, a
// foreign side of 10 units, so the creator alone set a price of 1e8 uqor per
// unit, and the running average has caught up with it.
func thinForeignPool(createdAt int64) ammtypes.Pool {
	return ammtypes.Pool{
		ID: 7, Type: ammtypes.PoolTypeConstantProduct, Status: ammtypes.PoolStatusActive,
		TokenA: "ibc/FOO", TokenB: "uqor", CreatedAt: createdAt,
		ReserveA: math.NewInt(10), ReserveB: math.NewInt(1_000_000_000),
		WeightedAvgPrice: math.LegacyNewDec(100_000_000),
	}
}

func twapToken() AcceptedFeeToken {
	return AcceptedFeeToken{
		Denom: "ibc/FOO", ConversionRate: "1", PriceSource: PriceSourceAMMTWAP,
		PoolID: 7, MinLiquidity: "100000000", MaxDeviationBps: 500,
	}
}

func cfgFor(tok AcceptedFeeToken) GasAbstractionConfig {
	c := DefaultGasAbstractionConfig()
	c.AcceptedTokens = append(c.AcceptedTokens, tok)
	return c
}

func ctxAtHeight(h int64) sdk.Context {
	return sdk.Context{}.WithBlockHeader(cmtproto.Header{Height: h})
}

func TestDepthAgeGateOffKeepsHistoricalPricing(t *testing.T) {
	depthAgeGate(t, false)
	tok := twapToken()
	rate, err := ResolveRate(ctxAtHeight(101), cfgFor(tok), tok, onePool{thinForeignPool(100)})
	if err != nil {
		t.Fatalf("historical rules priced this pool, got %v", err)
	}
	if !rate.Equal(math.LegacyNewDec(100_000_000)) {
		t.Fatalf("rate %s", rate)
	}
}

func TestDepthAgeGateRefusesWithoutForeignMinimum(t *testing.T) {
	depthAgeGate(t, true)
	tok := twapToken()
	_, err := ResolveRate(ctxAtHeight(1_000_000), cfgFor(tok), tok, onePool{thinForeignPool(100)})
	if err == nil || !strings.Contains(err.Error(), "min_foreign_liquidity") {
		t.Fatalf("want a refusal naming min_foreign_liquidity, got %v", err)
	}
}

func TestDepthAgeGateRefusesThinForeignSide(t *testing.T) {
	depthAgeGate(t, true)
	tok := twapToken()
	tok.MinForeignLiquidity = "1000000"
	_, err := ResolveFeeRate(ctxAtHeight(1_000_000), cfgFor(tok), tok, onePool{thinForeignPool(100)})
	if err == nil || !strings.Contains(err.Error(), "minimum foreign liquidity") {
		t.Fatalf("want the foreign-depth refusal, got %v", err)
	}
}

func deepPool(createdAt int64) ammtypes.Pool {
	p := thinForeignPool(createdAt)
	p.ReserveA = math.NewInt(2_000_000_000)
	p.ReserveB = math.NewInt(1_000_000_000)
	p.WeightedAvgPrice = math.LegacyMustNewDecFromStr("0.5")
	return p
}

func TestDepthAgeGateRefusesYoungPool(t *testing.T) {
	depthAgeGate(t, true)
	tok := twapToken()
	tok.MinForeignLiquidity = "1000000"
	h := int64(100) + MinFeePricingPoolAgeBlocks - 1
	if _, err := ResolveRate(ctxAtHeight(h), cfgFor(tok), tok, onePool{deepPool(100)}); err == nil || !strings.Contains(err.Error(), "younger") {
		t.Fatalf("want the age refusal one block early, got %v", err)
	}
	rate, err := ResolveRate(ctxAtHeight(h+1), cfgFor(tok), tok, onePool{deepPool(100)})
	if err != nil {
		t.Fatalf("aged, deep pool refused: %v", err)
	}
	if !rate.Equal(math.LegacyMustNewDecFromStr("0.5")) {
		t.Fatalf("rate %s", rate)
	}
	// A listing may ask for more than the floor, never less.
	tok.MinPoolAgeBlocks = MinFeePricingPoolAgeBlocks * 2
	if _, err := ResolveRate(ctxAtHeight(h+1), cfgFor(tok), tok, onePool{deepPool(100)}); err == nil {
		t.Fatal("listing age above the floor not applied")
	}
	tok.MinPoolAgeBlocks = 1
	if _, err := ResolveRate(ctxAtHeight(h), cfgFor(tok), tok, onePool{deepPool(100)}); err == nil {
		t.Fatal("listing age below the floor lowered it")
	}
}

// From the switch the refill path (ResolveRate) refuses a StableSwap pool too.
func TestDepthAgeGateConstantProductOnlyOnEveryPath(t *testing.T) {
	tok := twapToken()
	tok.MinForeignLiquidity = "1000000"
	p := deepPool(100)
	p.Type = ammtypes.PoolTypeStableSwap
	h := 100 + MinFeePricingPoolAgeBlocks
	depthAgeGate(t, false)
	if _, err := ResolveRate(ctxAtHeight(h), cfgFor(tok), tok, onePool{p}); err != nil {
		t.Fatalf("historical refill path refused a stable pool: %v", err)
	}
	depthAgeGate(t, true)
	if _, err := ResolveRate(ctxAtHeight(h), cfgFor(tok), tok, onePool{p}); err == nil || !strings.Contains(err.Error(), "constant_product") {
		t.Fatalf("want the pool-type refusal, got %v", err)
	}
}

func TestValidateNewListingFields(t *testing.T) {
	tok := twapToken()
	tok.MinForeignLiquidity = "0"
	if err := cfgFor(tok).Validate(); err == nil {
		t.Fatal("zero min_foreign_liquidity accepted")
	}
	tok.MinForeignLiquidity = "abc"
	if err := cfgFor(tok).Validate(); err == nil {
		t.Fatal("non-numeric min_foreign_liquidity accepted")
	}
	tok.MinForeignLiquidity = "5"
	tok.MinPoolAgeBlocks = -1
	if err := cfgFor(tok).Validate(); err == nil {
		t.Fatal("negative min_pool_age_blocks accepted")
	}
	tok.MinPoolAgeBlocks = 0
	if err := cfgFor(tok).Validate(); err != nil {
		t.Fatal(err)
	}
}

// A config stored before the new fields serialises to the same bytes.
func TestStoredConfigBytesUnchangedWhenFieldsEmpty(t *testing.T) {
	tok := twapToken()
	bz, err := json.Marshal(cfgFor(tok))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bz), "min_foreign_liquidity") || strings.Contains(string(bz), "min_pool_age_blocks") {
		t.Fatalf("empty new fields serialised: %s", bz)
	}
}
