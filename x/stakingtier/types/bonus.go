package types

import "cosmossdk.io/math"

// SecondsPerYear is the year every yearly rate is stated against: 365 days.
const SecondsPerYear = int64(365 * 24 * 60 * 60)

// The bonus arithmetic. Integer and LegacyDec only: it runs in consensus.
//
// WHAT EACH BONUS APPLIES TO (decided 2026-10-18):
//
//   - The lock bonus (3/6/12 months: +1.5/+3/+5 points) applies to the locked
//     amount.
//   - The light node bonus (+2) applies to the holder's locked amount. Only
//     locked stake earns a bonus, so a light node operator locks to get it.
//   - The validator bonus (+3) applies to the operator's own self-bond while
//     the validator is bonded and not jailed, locked or not. On the locked part
//     of a self-bond it stacks with the lock bonus (and the light node bonus);
//     the unlocked part earns the validator bonus alone.
//   - Base yield plus bonus never exceeds max_total_apy_bps (25% net): the
//     bonus of each stake is cut to the room left above its base yield.

// BaseAPYBps is the gross yearly yield, in basis points of the bonded stake,
// that baseEmission minted over periodSeconds gives stakers when stakerReach
// of it reaches them through the fee split:
//
//	baseEmission x stakerReach x 10,000 x SecondsPerYear / periodSeconds / bonded
//
// Zero for any non-positive input.
func BaseAPYBps(baseEmission, bonded math.Int, stakerReach math.LegacyDec, periodSeconds int64) math.LegacyDec {
	if baseEmission.IsNil() || !baseEmission.IsPositive() || bonded.IsNil() || !bonded.IsPositive() ||
		stakerReach.IsNil() || !stakerReach.IsPositive() || periodSeconds <= 0 {
		return math.LegacyZeroDec()
	}
	return math.LegacyNewDecFromInt(baseEmission).
		Mul(stakerReach).
		MulInt64(BasisPoints).
		MulInt64(SecondsPerYear).
		QuoInt64(periodSeconds).
		QuoInt(bonded)
}

// NetOfCommission is the base yield a delegator keeps after the validator's
// commission. A commission outside [0, 1] is clamped.
func NetOfCommission(gross, commission math.LegacyDec) math.LegacyDec {
	if gross.IsNil() || !gross.IsPositive() {
		return math.LegacyZeroDec()
	}
	if commission.IsNil() || commission.IsNegative() {
		return gross
	}
	if commission.GT(math.LegacyOneDec()) {
		return math.LegacyZeroDec()
	}
	return gross.Mul(math.LegacyOneDec().Sub(commission))
}

// CappedBonusBps is the bonus a stake actually earns: bonusBps, cut so that
// baseBps plus the bonus does not exceed maxTotalBps. Never negative.
func CappedBonusBps(bonusBps uint32, baseBps math.LegacyDec, maxTotalBps uint32) math.LegacyDec {
	bonus := math.LegacyNewDec(int64(bonusBps))
	if baseBps.IsNil() || baseBps.IsNegative() {
		baseBps = math.LegacyZeroDec()
	}
	room := math.LegacyNewDec(int64(maxTotalBps)).Sub(baseBps)
	if !room.IsPositive() {
		return math.LegacyZeroDec()
	}
	if bonus.GT(room) {
		return room
	}
	return bonus
}

// LockBonusBps is the uncapped bonus of a lock: its term's bonus, plus the
// light node bonus when the holder runs an active light node, plus the
// validator bonus when the lock is part of an active validator's self-bond.
func LockBonusBps(p Params, tierBps uint32, lightNode, selfBondOfActiveValidator bool) uint32 {
	rate := tierBps
	if lightNode {
		rate += p.LightNodeBonusBps
	}
	if selfBondOfActiveValidator {
		rate += p.ValidatorBonusBps
	}
	return rate
}

// BonusOwed is what tokens earn at rateBps a year over seconds, truncated to
// whole uqor.
func BonusOwed(tokens math.Int, rateBps math.LegacyDec, seconds int64) math.Int {
	if tokens.IsNil() || !tokens.IsPositive() || rateBps.IsNil() || !rateBps.IsPositive() || seconds <= 0 {
		return math.ZeroInt()
	}
	return math.LegacyNewDecFromInt(tokens).
		Mul(rateBps).
		MulInt64(seconds).
		QuoInt64(BasisPoints).
		QuoInt64(SecondsPerYear).
		TruncateInt()
}

// ProRata scales one bonus by funded/totalOwed when the epoch's funding falls
// short, truncated. Every bonus is scaled by the same factor, so the shortfall
// is shared in proportion. owed unchanged when funding covers the total.
func ProRata(owed, funded, totalOwed math.Int) math.Int {
	if owed.IsNil() || !owed.IsPositive() || funded.IsNil() || !funded.IsPositive() ||
		totalOwed.IsNil() || !totalOwed.IsPositive() {
		return math.ZeroInt()
	}
	if funded.GTE(totalOwed) {
		return owed
	}
	return owed.Mul(funded).Quo(totalOwed)
}
