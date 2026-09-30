package types

import (
	"math"
	"strings"
	"testing"
)

// The reported vector (QSR-2026-0012): InputSize = 2^62 with one hidden layer of
// four wraps the int parameter count to a small number, so a policy with a
// handful of weights satisfies the legacy length check.
func reporterConfig() MLPConfig {
	return MLPConfig{InputSize: 1 << 62, HiddenSizes: []int{4}, OutputSize: ActionDimensions}
}

func TestLegacyTotalParamsStillWraps(t *testing.T) {
	// ValidateBasic keeps the legacy arithmetic so replay is unchanged; this
	// pins that the legacy value is the wrapped one, not a quiet fix.
	c := reporterConfig()
	got := c.TotalParams()
	want := 4 + 4*ActionDimensions + ActionDimensions // 2^64 wraps to 0
	if got != want {
		t.Fatalf("legacy TotalParams = %d, want the wrapped %d", got, want)
	}
	pw := PolicyWeights{Config: c, Weights: make([]int64, got)}
	if err := pw.Validate(); err != nil {
		t.Fatalf("legacy Validate must still accept the wrapped vector (replay): %v", err)
	}
}

func TestTotalParamsCheckedRefusesOverflow(t *testing.T) {
	if _, err := reporterConfig().TotalParamsChecked(); err == nil {
		t.Fatal("expected an overflow error")
	}
	big := MLPConfig{InputSize: math.MaxInt64, HiddenSizes: []int{math.MaxInt64}, OutputSize: 1}
	if _, err := big.TotalParamsChecked(); err == nil {
		t.Fatal("expected an overflow error for MaxInt64 layers")
	}
	neg := MLPConfig{InputSize: -1, HiddenSizes: []int{4}, OutputSize: 5}
	if _, err := neg.TotalParamsChecked(); err == nil {
		t.Fatal("expected an error for a negative size")
	}
}

func TestDefaultPolicyIsAccepted(t *testing.T) {
	c := DefaultMLPConfig()
	n, err := c.TotalParamsChecked()
	if err != nil || n != 73_733 {
		t.Fatalf("default config: %d, %v; want 73,733", n, err)
	}
	if uint64(c.TotalParams()) != n {
		t.Fatal("checked and legacy counts must agree on a sane config")
	}
	pw := PolicyWeights{Config: c, Weights: make([]int64, n)}
	pw.Weights[0] = MaxPolicyWeightAbs
	pw.Weights[1] = -MaxPolicyWeightAbs
	if err := pw.ValidateStrict(); err != nil {
		t.Fatalf("default policy refused: %v", err)
	}
}

func TestValidateStrictBounds(t *testing.T) {
	ok := DefaultMLPConfig()
	cases := []struct {
		name string
		mut  func(*PolicyWeights)
		want string
	}{
		{"reporter vector", func(p *PolicyWeights) { p.Config = reporterConfig(); p.Weights = make([]int64, 29) }, "input size"},
		{"wrong input", func(p *PolicyWeights) { p.Config.InputSize = 24 }, "input size"},
		{"wrong output", func(p *PolicyWeights) { p.Config.OutputSize = 6 }, "output size"},
		{"no hidden", func(p *PolicyWeights) { p.Config.HiddenSizes = nil }, "hidden layers"},
		{"too many layers", func(p *PolicyWeights) { p.Config.HiddenSizes = []int{8, 8, 8, 8, 8} }, "hidden layers"},
		{"too wide", func(p *PolicyWeights) { p.Config.HiddenSizes = []int{1025} }, "width"},
		{"zero width", func(p *PolicyWeights) { p.Config.HiddenSizes = []int{0} }, "width"},
		{"too many params", func(p *PolicyWeights) { p.Config.HiddenSizes = []int{1024, 1024, 1024} }, "parameters"},
		{"short weights", func(p *PolicyWeights) { p.Weights = p.Weights[1:] }, "length"},
		{"weight too large", func(p *PolicyWeights) { p.Weights[7] = MaxPolicyWeightAbs + 1 }, "outside"},
		{"weight too small", func(p *PolicyWeights) { p.Weights[7] = -MaxPolicyWeightAbs - 1 }, "outside"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, _ := ok.TotalParamsChecked()
			pw := PolicyWeights{Config: MLPConfig{InputSize: ok.InputSize, HiddenSizes: append([]int(nil), ok.HiddenSizes...), OutputSize: ok.OutputSize}, Weights: make([]int64, n)}
			tc.mut(&pw)
			err := pw.ValidateStrict()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

func FuzzTotalParamsChecked(f *testing.F) {
	f.Add(int64(25), int64(256), int64(256), int64(5))
	f.Add(int64(1<<62), int64(4), int64(0), int64(5))
	f.Fuzz(func(t *testing.T, in, h1, h2, out int64) {
		c := MLPConfig{InputSize: int(in), HiddenSizes: []int{int(h1), int(h2)}, OutputSize: int(out)}
		n, err := c.TotalParamsChecked() // must never panic
		if err != nil || in < 0 || h1 < 0 || h2 < 0 || out < 0 {
			return
		}
		// Cross-check small results against the legacy arithmetic, which is
		// exact when nothing overflows.
		if n < 1<<40 && in < 1<<20 && h1 < 1<<20 && h2 < 1<<20 && out < 1<<20 {
			if uint64(c.TotalParams()) != n {
				t.Fatalf("checked %d != legacy %d for %+v", n, c.TotalParams(), c)
			}
		}
	})
}
