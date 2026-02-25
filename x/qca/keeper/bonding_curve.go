//go:build proprietary

package keeper

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/mathutil"
)

// CalculateBondingReward computes the bonding curve reward for a validator.
// R(v,t) = beta * S_v * (1 + alpha * log(1 + L_v)) * Q(r_v) * P(t)
func (k Keeper) CalculateBondingReward(
	ctx sdk.Context,
	selfBondedStake uint64,
	loyaltyBlocks int64,
	reputationScore float64,
) (math.LegacyDec, error) {
	config := k.GetConfig(ctx)
	bcConfig := config.BondingCurveConfig

	// Parse config values
	alpha, err := math.LegacyNewDecFromStr(bcConfig.Alpha)
	if err != nil {
		return math.LegacyZeroDec(), fmt.Errorf("invalid alpha: %w", err)
	}
	beta, err := math.LegacyNewDecFromStr(bcConfig.Beta)
	if err != nil {
		return math.LegacyZeroDec(), fmt.Errorf("invalid beta: %w", err)
	}
	phaseMul, err := math.LegacyNewDecFromStr(bcConfig.PhaseMultiplier)
	if err != nil {
		return math.LegacyZeroDec(), fmt.Errorf("invalid phase_multiplier: %w", err)
	}

	// S_v -- self-bonded stake
	sv := math.LegacyNewDec(int64(selfBondedStake))

	// L_v -- loyalty duration (blocks)
	lv := math.LegacyNewDec(loyaltyBlocks)

	// log(1 + L_v) using deterministic Taylor series
	logLoyalty := mathutil.TaylorLn1PlusX(lv)

	// Loyalty factor: 1 + alpha * log(1 + L_v)
	one := math.LegacyOneDec()
	loyaltyFactor := one.Add(alpha.Mul(logLoyalty))

	// Q(r_v) = 1 + 0.5 * (r_v - 0.5), clamped to [0.75, 1.25]
	rv, err := math.LegacyNewDecFromStr(fmt.Sprintf("%.18f", reputationScore))
	if err != nil {
		rv = math.LegacyMustNewDecFromStr("0.5") // default to neutral
	}
	halfDec := math.LegacyMustNewDecFromStr("0.5")
	qrv := one.Add(halfDec.Mul(rv.Sub(halfDec)))
	// Clamp Q to [0.75, 1.25]
	minQ := math.LegacyMustNewDecFromStr("0.75")
	maxQ := math.LegacyMustNewDecFromStr("1.25")
	if qrv.LT(minQ) {
		qrv = minQ
	}
	if qrv.GT(maxQ) {
		qrv = maxQ
	}

	// R(v,t) = beta * S_v * loyaltyFactor * Q(r_v) * P(t)
	reward := beta.Mul(sv).Mul(loyaltyFactor).Mul(qrv).Mul(phaseMul)

	// Update stats
	stats := k.GetStats(ctx)
	stats.BondingCalculations++
	k.SetStats(ctx, stats)

	return reward, nil
}
