package mathutil

import (
	"testing"

	sdkmath "cosmossdk.io/math"
)

// helper: build a LegacyDec from a string literal.
func dec(s string) sdkmath.LegacyDec {
	return sdkmath.LegacyMustNewDecFromStr(s)
}

// helper: assert |got - want| <= tolerance.
func assertApprox(t *testing.T, name string, got, want, tol sdkmath.LegacyDec) {
	t.Helper()
	diff := got.Sub(want).Abs()
	if diff.GT(tol) {
		t.Errorf("%s: got %s, want %s (diff %s > tol %s)", name, got, want, diff, tol)
	}
}

// ---------------------------------------------------------------------------
// IntegerSqrt
// ---------------------------------------------------------------------------

func TestIntegerSqrt_Zero(t *testing.T) {
	assertApprox(t, "sqrt(0)", IntegerSqrt(dec("0")), dec("0"), dec("0.000000000001"))
}

func TestIntegerSqrt_One(t *testing.T) {
	assertApprox(t, "sqrt(1)", IntegerSqrt(dec("1")), dec("1"), dec("0.000000000001"))
}

func TestIntegerSqrt_Four(t *testing.T) {
	assertApprox(t, "sqrt(4)", IntegerSqrt(dec("4")), dec("2"), dec("0.000000000001"))
}

func TestIntegerSqrt_Nine(t *testing.T) {
	assertApprox(t, "sqrt(9)", IntegerSqrt(dec("9")), dec("3"), dec("0.000000000001"))
}

func TestIntegerSqrt_Two(t *testing.T) {
	// sqrt(2) = 1.41421356237309504...
	assertApprox(t, "sqrt(2)", IntegerSqrt(dec("2")), dec("1.414213562373095"), dec("0.000000000001"))
}

func TestIntegerSqrt_Hundred(t *testing.T) {
	assertApprox(t, "sqrt(100)", IntegerSqrt(dec("100")), dec("10"), dec("0.000000000001"))
}

func TestIntegerSqrt_Quarter(t *testing.T) {
	assertApprox(t, "sqrt(0.25)", IntegerSqrt(dec("0.25")), dec("0.5"), dec("0.000000000001"))
}

func TestIntegerSqrt_Million(t *testing.T) {
	assertApprox(t, "sqrt(1000000)", IntegerSqrt(dec("1000000")), dec("1000"), dec("0.000000000001"))
}

// ---------------------------------------------------------------------------
// TaylorLn1PlusX
// ---------------------------------------------------------------------------

func TestTaylorLn1PlusX_Zero(t *testing.T) {
	assertApprox(t, "ln(1+0)", TaylorLn1PlusX(dec("0")), dec("0"), dec("0.0001"))
}

func TestTaylorLn1PlusX_Half(t *testing.T) {
	// ln(1.5) = 0.405465...
	assertApprox(t, "ln(1+0.5)", TaylorLn1PlusX(dec("0.5")), dec("0.405465"), dec("0.0001"))
}

func TestTaylorLn1PlusX_One(t *testing.T) {
	// ln(2) = 0.693147...
	assertApprox(t, "ln(1+1)", TaylorLn1PlusX(dec("1.0")), dec("0.693147"), dec("0.0001"))
}

func TestTaylorLn1PlusX_Large(t *testing.T) {
	// ln(7.389) ~ 2.000
	assertApprox(t, "ln(1+6.389)", TaylorLn1PlusX(dec("6.389")), dec("2.0"), dec("0.01"))
}

func TestTaylorLn1PlusX_Negative(t *testing.T) {
	// Negative input should return 0 (invalid).
	result := TaylorLn1PlusX(dec("-0.5"))
	if !result.IsZero() {
		t.Errorf("TaylorLn1PlusX(-0.5): expected 0, got %s", result)
	}
}

// ---------------------------------------------------------------------------
// SigmoidApprox
// ---------------------------------------------------------------------------

func TestSigmoidApprox_Zero(t *testing.T) {
	assertApprox(t, "sigmoid(0)", SigmoidApprox(dec("0")), dec("0.5"), dec("0.001"))
}

func TestSigmoidApprox_Positive3(t *testing.T) {
	assertApprox(t, "sigmoid(3)", SigmoidApprox(dec("3")), dec("0.9526"), dec("0.01"))
}

func TestSigmoidApprox_Negative3(t *testing.T) {
	assertApprox(t, "sigmoid(-3)", SigmoidApprox(dec("-3")), dec("0.0474"), dec("0.01"))
}

func TestSigmoidApprox_Positive1(t *testing.T) {
	assertApprox(t, "sigmoid(1)", SigmoidApprox(dec("1")), dec("0.7311"), dec("0.01"))
}

func TestSigmoidApprox_Negative1(t *testing.T) {
	assertApprox(t, "sigmoid(-1)", SigmoidApprox(dec("-1")), dec("0.2689"), dec("0.01"))
}

func TestSigmoidApprox_Symmetry(t *testing.T) {
	vals := []string{"0.5", "1.0", "1.5", "2.0", "2.5", "3.0"}
	for _, v := range vals {
		pos := SigmoidApprox(dec(v))
		neg := SigmoidApprox(dec("-" + v))
		sum := pos.Add(neg)
		assertApprox(t, "symmetry("+v+")", sum, dec("1.0"), dec("0.001"))
	}
}

// ---------------------------------------------------------------------------
// ReputationMultiplier
// ---------------------------------------------------------------------------

func TestReputationMultiplier_Zero(t *testing.T) {
	// 0.5 + 1.5 * sigmoid(-3) = 0.5 + 1.5*0.0474 ~ 0.571
	assertApprox(t, "repMul(0)", ReputationMultiplier(dec("0")), dec("0.571"), dec("0.05"))
}

func TestReputationMultiplier_Half(t *testing.T) {
	assertApprox(t, "repMul(0.5)", ReputationMultiplier(dec("0.5")), dec("1.25"), dec("0.05"))
}

func TestReputationMultiplier_One(t *testing.T) {
	// 0.5 + 1.5 * sigmoid(3) = 0.5 + 1.5*0.9526 ~ 1.929
	assertApprox(t, "repMul(1.0)", ReputationMultiplier(dec("1.0")), dec("1.929"), dec("0.05"))
}

func TestReputationMultiplier_Quarter(t *testing.T) {
	result := ReputationMultiplier(dec("0.25"))
	lower := dec("0.5")
	upper := dec("1.25")
	if result.LT(lower) || result.GT(upper) {
		t.Errorf("repMul(0.25) = %s, want in [0.5, 1.25]", result)
	}
}

func TestReputationMultiplier_ThreeQuarters(t *testing.T) {
	result := ReputationMultiplier(dec("0.75"))
	lower := dec("1.25")
	upper := dec("2.0")
	if result.LT(lower) || result.GT(upper) {
		t.Errorf("repMul(0.75) = %s, want in [1.25, 2.0]", result)
	}
}

func TestReputationMultiplier_Clamped(t *testing.T) {
	// Even at extreme inputs the result stays in [0.5, 2.0].
	low := ReputationMultiplier(dec("-1"))
	high := ReputationMultiplier(dec("5"))
	if low.LT(dec("0.5")) {
		t.Errorf("repMul(-1) = %s, should be >= 0.5", low)
	}
	if high.GT(dec("2.0")) {
		t.Errorf("repMul(5) = %s, should be <= 2.0", high)
	}
}

// ---------------------------------------------------------------------------
// ExpApprox
// ---------------------------------------------------------------------------

func TestExpApprox_Zero(t *testing.T) {
	assertApprox(t, "exp(0)", ExpApprox(dec("0")), dec("1.0"), dec("0.001"))
}

func TestExpApprox_One(t *testing.T) {
	assertApprox(t, "exp(1)", ExpApprox(dec("1")), dec("2.71828"), dec("0.001"))
}

func TestExpApprox_NegOne(t *testing.T) {
	assertApprox(t, "exp(-1)", ExpApprox(dec("-1")), dec("0.36788"), dec("0.001"))
}

func TestExpApprox_Half(t *testing.T) {
	assertApprox(t, "exp(0.5)", ExpApprox(dec("0.5")), dec("1.64872"), dec("0.001"))
}

func TestExpApprox_NegLn2(t *testing.T) {
	// exp(-ln2) = 0.5
	assertApprox(t, "exp(-ln2)", ExpApprox(dec("-0.693147")), dec("0.5"), dec("0.001"))
}
