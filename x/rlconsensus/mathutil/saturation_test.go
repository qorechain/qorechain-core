package mathutil

import (
	"testing"

	sdkmath "cosmossdk.io/math"
)

// These functions run inside BeginBlock and EndBlock, where a panic is not a
// failed transaction but a halted chain. Whatever the input, they must return.
func TestExpApproxSaturatesInsteadOfPanicking(t *testing.T) {
	for _, x := range []int64{170, 177, 200, 1000, 1 << 40} {
		got := ExpApprox(sdkmath.LegacyNewDec(x))
		want := ExpApprox(ExpArgumentBound)
		if !got.Equal(want) {
			t.Fatalf("ExpApprox(%d) = %s, want the saturated value %s", x, got, want)
		}
		if !got.IsPositive() {
			t.Fatalf("ExpApprox(%d) is not positive", x)
		}
	}
	if !ExpApprox(sdkmath.LegacyNewDec(-1000)).Equal(ExpApprox(ExpArgumentBound.Neg())) {
		t.Fatal("large negative arguments must saturate too")
	}
	// Inside the bound the function is untouched.
	if !ExpApprox(sdkmath.LegacyNewDec(1)).Sub(sdkmath.LegacyMustNewDecFromStr("2.718281828")).Abs().LT(sdkmath.LegacyMustNewDecFromStr("0.000001")) {
		t.Fatal("exp(1) drifted")
	}
}

func TestSigmoidApproxIsBoundedForEveryInput(t *testing.T) {
	for _, x := range []int64{-1000000, -200, -177, -41, -40, -3, 0, 3, 40, 41, 177, 200, 1000000} {
		got := SigmoidApprox(sdkmath.LegacyNewDec(x))
		if got.IsNegative() || got.GT(sdkmath.LegacyOneDec()) {
			t.Fatalf("sigmoid(%d) = %s is outside [0, 1]", x, got)
		}
	}
	if !SigmoidApprox(sdkmath.LegacyNewDec(200)).Equal(sdkmath.LegacyOneDec()) {
		t.Fatal("sigmoid(200) must be 1")
	}
	if !SigmoidApprox(sdkmath.LegacyNewDec(-200)).IsZero() {
		t.Fatal("sigmoid(-200) must be 0")
	}
}
