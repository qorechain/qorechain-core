package types

import (
	"strings"
	"testing"
)

// ToFixedPoint runs in the block hooks. A dimension beyond the int64 range must
// come back as an error, not as a panic that stops every validator.
func TestToFixedPointRejectsOutOfRangeValuesWithoutPanicking(t *testing.T) {
	var o Observation
	for i := range o.Values {
		o.Values[i] = "0.5"
	}
	if _, err := o.ToFixedPoint(); err != nil {
		t.Fatalf("in-range observation: %v", err)
	}
	for _, v := range []string{"92233720369", "1000000000000000000", "-92233720369"} {
		o.Values[3] = v
		_, err := o.ToFixedPoint()
		if err == nil {
			t.Fatalf("value %s converted without error", v)
		}
		if !strings.Contains(err.Error(), "observation[3]") {
			t.Fatalf("error does not name the dimension: %v", err)
		}
	}
	// The largest representable value still converts.
	o.Values[3] = "92233720368"
	if _, err := o.ToFixedPoint(); err != nil {
		t.Fatalf("largest in-range value refused: %v", err)
	}
}

// Values large enough to make LegacyDec.Mul itself panic (its product passes
// the 256-bit bound) are refused before the multiplication, and the boundary
// is unchanged: the largest value whose truncated scaled form fits int64 still
// converts to max int64, the next one is refused.
func TestToFixedPointRangeCheckedBeforeMultiplying(t *testing.T) {
	var o Observation
	for i := range o.Values {
		o.Values[i] = "0"
	}
	huge := "1" + strings.Repeat("0", 72) // 1e72 parses; 1e72 * 1e8 overflows LegacyDec.Mul
	for _, v := range []string{huge, "-" + huge} {
		o.Values[2] = v
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("value %s panicked: %v", v, r)
				}
			}()
			if _, err := o.ToFixedPoint(); err == nil || !strings.Contains(err.Error(), "observation[2]") {
				t.Fatalf("value %s: err = %v", v, err)
			}
		}()
	}

	o.Values[2] = "92233720368.547758079999999999"
	out, err := o.ToFixedPoint()
	if err != nil || out[2] != 9223372036854775807 {
		t.Fatalf("largest convertible value: out=%d err=%v", out[2], err)
	}
	o.Values[2] = "92233720368.54775808"
	if _, err := o.ToFixedPoint(); err == nil {
		t.Fatal("the first out-of-range value converted")
	}
	o.Values[2] = "-92233720368.547758079999999999"
	if out, err := o.ToFixedPoint(); err != nil || out[2] != -9223372036854775807 {
		t.Fatalf("most negative convertible value: out=%d err=%v", out[2], err)
	}
}
