package types

import (
	"testing"

	"cosmossdk.io/math"
)

func TestUptimeFactor(t *testing.T) {
	cases := []struct {
		total, expected uint64
		want            math.LegacyDec
	}{
		{0, 0, math.LegacyOneDec()},
		{8, 10, math.LegacyNewDecWithPrec(8, 1)},
		{12, 10, math.LegacyOneDec()},
		{0, 10, math.LegacyZeroDec()},
	}
	for _, c := range cases {
		n := LightNodeInfo{TotalHeartbeats: c.total, ExpectedHeartbeats: c.expected}
		if got := n.UptimeFactor(); !got.Equal(c.want) {
			t.Fatalf("%d/%d: got %s want %s", c.total, c.expected, got, c.want)
		}
	}
}
