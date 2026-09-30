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
