package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

var bpsDenom = math.LegacyNewDec(10_000)

// ResolveRate returns the native units one unit of token.Denom is worth now,
// or an error saying why the token cannot pay fees at this height.
func ResolveRate(ctx sdk.Context, cfg GasAbstractionConfig, token AcceptedFeeToken, pools PoolReader) (math.LegacyDec, error) {
	return resolveRate(ctx, cfg, token, pools, false)
}

// ResolveFeeRate is ResolveRate under the gasabstraction-pool-type-pricing
// switch (QSR-2026-0634): an amm_twap token is priced only from a
// constant_product pool. The pool's average is a running average of its
// reserve ratio, which is a constant-product pool's price but not a StableSwap
// pool's: a StableSwap curve is flat, so its reserves can be skewed far at
// little cost while its real price stays near 1, and the average then follows
// the skewed ratio. Any other pool type is refused. The reads are those of
// ResolveRate, so the switch adds no gas here.
func ResolveFeeRate(ctx sdk.Context, cfg GasAbstractionConfig, token AcceptedFeeToken, pools PoolReader) (math.LegacyDec, error) {
	return resolveRate(ctx, cfg, token, pools, true)
}

func resolveRate(ctx sdk.Context, cfg GasAbstractionConfig, token AcceptedFeeToken, pools PoolReader, constantProductOnly bool) (math.LegacyDec, error) {
	switch token.PriceSource {
	case PriceSourceStatic:
		if token.RateExpiryHeight <= 0 || ctx.BlockHeight() > token.RateExpiryHeight {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf(
				"the static rate for %s expired at height %d", token.Denom, token.RateExpiryHeight)
		}
		rate, err := math.LegacyNewDecFromStr(token.ConversionRate)
		if err != nil || !rate.IsPositive() {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf("invalid static rate for %s", token.Denom)
		}
		return rate, nil

	case PriceSourceAMMTWAP:
		if pools == nil {
			return math.LegacyDec{}, ErrConversionFailed.Wrap("no pool reader")
		}
		pool, ok := pools.GetPool(ctx, token.PoolID)
		if !ok || pool.Status != ammtypes.PoolStatusActive {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf("pool %d is not active", token.PoolID)
		}
		if constantProductOnly && pool.Type != ammtypes.PoolTypeConstantProduct {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf(
				"pool %d is a %s pool; fees are priced only from constant_product pools", token.PoolID, pool.Type)
		}
		var foreignReserve, nativeReserve math.Int
		switch {
		case pool.TokenA == token.Denom && pool.TokenB == cfg.NativeDenom:
			foreignReserve, nativeReserve = pool.ReserveA, pool.ReserveB
		case pool.TokenB == token.Denom && pool.TokenA == cfg.NativeDenom:
			foreignReserve, nativeReserve = pool.ReserveB, pool.ReserveA
		default:
			return math.LegacyDec{}, ErrConversionFailed.Wrapf(
				"pool %d does not pair %s with %s", token.PoolID, token.Denom, cfg.NativeDenom)
		}
		minLiq, ok := math.NewIntFromString(token.MinLiquidity)
		if !ok || foreignReserve.IsNil() || nativeReserve.IsNil() || !foreignReserve.IsPositive() || nativeReserve.LT(minLiq) {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf(
				"pool %d is below the minimum liquidity for fee pricing", token.PoolID)
		}
		twap := pool.WeightedAvgPrice // TokenB per TokenA
		if twap.IsNil() || !twap.IsPositive() {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf("pool %d has no average price yet", token.PoolID)
		}
		if pool.TokenB == token.Denom {
			twap = math.LegacyOneDec().Quo(twap)
		}
		spot := math.LegacyNewDecFromInt(nativeReserve).Quo(math.LegacyNewDecFromInt(foreignReserve))
		maxDev := math.LegacyNewDec(int64(token.MaxDeviationBps)).Quo(bpsDenom)
		if twap.Sub(spot).Abs().GT(spot.Mul(maxDev)) {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf(
				"pool %d spot price deviates from its average by more than %d bps", token.PoolID, token.MaxDeviationBps)
		}
		if !twap.IsPositive() {
			return math.LegacyDec{}, ErrConversionFailed.Wrapf("pool %d price rounds to zero", token.PoolID)
		}
		return twap, nil

	default:
		return math.LegacyDec{}, ErrUnsupportedDenom.Wrapf(
			"denom %s is listed as accepted for fees but has no price source; pay fees in %s",
			token.Denom, cfg.NativeDenom)
	}
}

// NativeValue is what amount of the foreign token is worth to the float:
// floor(amount x rate x (1 - premium)).
func NativeValue(amount math.Int, rate math.LegacyDec, premiumBps uint32) math.Int {
	keep := bpsDenom.Sub(math.LegacyNewDec(int64(premiumBps))).Quo(bpsDenom)
	return math.LegacyNewDecFromInt(amount).Mul(rate).Mul(keep).TruncateInt()
}

// FeeValue is NativeValue under the gasabstraction-pool-type-pricing switch
// (QSR-2026-0634). For an amm_twap token the fee is valued at the lower of
// amount x rate and what the pool would really pay for the whole amount
// (QuoteExactIn, read-only, the swap's own pricing with its fee, slippage and
// impact cap), less the premium. The average says what one unit was worth
// over time; the quote says what this amount fetches now, and the float must
// not pay more than either. A quote the pool refuses refuses the fee. A static
// rate is valued as NativeValue.
func FeeValue(ctx sdk.Context, cfg GasAbstractionConfig, token AcceptedFeeToken, amount math.Int, rate math.LegacyDec, pools PoolReader) (math.Int, error) {
	if token.PriceSource != PriceSourceAMMTWAP {
		return NativeValue(amount, rate, cfg.PremiumBps), nil
	}
	quoter, ok := pools.(PoolQuoter)
	if !ok {
		return math.Int{}, ErrConversionFailed.Wrap("no pool quoter")
	}
	quote, _, err := quoter.QuoteExactIn(ctx, token.PoolID, token.Denom, amount)
	if err != nil {
		return math.Int{}, ErrConversionFailed.Wrapf("pool %d cannot quote %s%s: %s", token.PoolID, amount, token.Denom, err)
	}
	if quote.IsNil() || quote.IsNegative() {
		return math.Int{}, ErrConversionFailed.Wrapf("pool %d returned an invalid quote", token.PoolID)
	}
	base := math.LegacyNewDecFromInt(amount).Mul(rate)
	if q := math.LegacyNewDecFromInt(quote); q.LT(base) {
		base = q
	}
	keep := bpsDenom.Sub(math.LegacyNewDec(int64(cfg.PremiumBps))).Quo(bpsDenom)
	return base.Mul(keep).TruncateInt(), nil
}
