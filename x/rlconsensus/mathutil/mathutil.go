// Package mathutil provides deterministic math functions used by both
// x/rlconsensus and x/qca. All computations use cosmossdk.io/math.LegacyDec
// to avoid non-deterministic floating-point arithmetic.
package mathutil

import (
	sdkmath "cosmossdk.io/math"
)

var (
	zero = sdkmath.LegacyZeroDec()
	one  = sdkmath.LegacyOneDec()
	two  = sdkmath.LegacyNewDec(2)
	half = one.Quo(two)
)

// IntegerSqrt computes the square root of x using Newton's method.
// The computation is fully deterministic (no float64). Returns zero for x <= 0.
func IntegerSqrt(x sdkmath.LegacyDec) sdkmath.LegacyDec {
	if x.IsZero() || x.IsNegative() {
		return zero
	}

	// Initial guess: x/2 if x >= 1, otherwise 1.
	guess := x.Quo(two)
	if x.LT(one) {
		guess = one
	}

	epsilon := sdkmath.LegacyMustNewDecFromStr("0.000000000000000001") // 1e-18

	for i := 0; i < 100; i++ {
		// next = (guess + x/guess) / 2
		next := guess.Add(x.Quo(guess)).Quo(two)
		diff := next.Sub(guess).Abs()
		if diff.LT(epsilon) {
			return next
		}
		guess = next
	}
	return guess
}

// TaylorLn1PlusX computes ln(1+x) for x >= 0 using argument reduction
// followed by a Taylor series expansion. Returns zero for x < 0.
func TaylorLn1PlusX(x sdkmath.LegacyDec) sdkmath.LegacyDec {
	if x.IsZero() {
		return zero
	}
	if x.IsNegative() {
		return zero
	}

	val := one.Add(x)
	result := zero
	ln2 := sdkmath.LegacyMustNewDecFromStr("0.693147180559945309")

	// Argument reduction: divide val by 2 until val <= 1.5.
	// This keeps u = val-1 in [0, 0.5] where the Taylor series converges well.
	threeHalves := sdkmath.LegacyMustNewDecFromStr("1.5")
	for val.GT(threeHalves) {
		val = val.Quo(two)
		result = result.Add(ln2)
	}

	// Taylor series for ln(1 + u), where u = val - 1 and u is in (-1, 1].
	u := val.Sub(one)
	term := u    // u^n accumulator
	sum := zero

	for n := 1; n <= 15; n++ {
		nDec := sdkmath.LegacyNewDec(int64(n))
		contrib := term.Quo(nDec)
		if n%2 == 1 {
			sum = sum.Add(contrib)
		} else {
			sum = sum.Sub(contrib)
		}
		term = term.Mul(u)
	}

	return result.Add(sum)
}

// SigmoidApprox computes the sigmoid function using the Taylor exp approximation.
// sigmoid(x) = exp(x) / (1 + exp(x))
//
// For numerical stability with negative x, uses: sigmoid(x) = 1 - sigmoid(-x).
// Error < 0.1% for x in [-3, 3].
func SigmoidApprox(x sdkmath.LegacyDec) sdkmath.LegacyDec {
	// For negative x, use symmetry: sigmoid(x) = 1 - sigmoid(-x)
	if x.IsNegative() {
		return one.Sub(SigmoidApprox(x.Neg()))
	}

	ex := ExpApprox(x)
	return ex.Quo(one.Add(ex))
}

// ReputationMultiplier maps a reputation score r in [0, 1] to a multiplier
// in [0.5, 2.0] using a sigmoid curve.
//
// Formula: 0.5 + 1.5 * sigmoid(6 * (r - 0.5))
//
// The result is clamped to [0.5, 2.0].
func ReputationMultiplier(r sdkmath.LegacyDec) sdkmath.LegacyDec {
	six := sdkmath.LegacyNewDec(6)
	onePointFive := sdkmath.LegacyMustNewDecFromStr("1.5")
	minVal := sdkmath.LegacyMustNewDecFromStr("0.5")
	maxVal := sdkmath.LegacyNewDec(2)

	arg := six.Mul(r.Sub(half))
	sig := SigmoidApprox(arg)
	result := minVal.Add(onePointFive.Mul(sig))

	// Clamp to [0.5, 2.0].
	if result.LT(minVal) {
		return minVal
	}
	if result.GT(maxVal) {
		return maxVal
	}
	return result
}

// ExpApprox computes exp(x) using a Taylor series with 12 terms:
//
//	exp(x) = 1 + x + x^2/2! + x^3/3! + ... + x^12/12!
func ExpApprox(x sdkmath.LegacyDec) sdkmath.LegacyDec {
	sum := one
	term := one // term_n = x^n / n!

	for n := int64(1); n <= 12; n++ {
		nDec := sdkmath.LegacyNewDec(n)
		term = term.Mul(x).Quo(nDec)
		sum = sum.Add(term)
	}

	return sum
}
