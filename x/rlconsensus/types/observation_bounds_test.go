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
