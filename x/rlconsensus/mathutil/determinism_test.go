package mathutil

import (
	"sync"
	"testing"

	sdkmath "cosmossdk.io/math"
)

// TestExpApprox_Determinism is the regression test for v2.6.2.
// The reputation EndBlocker previously used math.Exp / math.Log1p
// (platform-dependent IEEE 754 in trailing bits), which caused
// validators on different machines to compute different reputation
// scores and disagree on block commitment hashes.
//
// All deterministic math functions must return bit-identical results
// for identical inputs across many invocations and across goroutines.
func TestExpApprox_Determinism(t *testing.T) {
	inputs := []string{
		"-2.5",
		"-1.0",
		"-0.5",
		"-0.001",
		"0",
		"0.001",
		"0.5",
		"1.0",
		"2.5",
	}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			x := sdkmath.LegacyMustNewDecFromStr(in)
			ref := ExpApprox(x)
			for i := 0; i < 256; i++ {
				if got := ExpApprox(x); !got.Equal(ref) {
					t.Fatalf("ExpApprox(%s) iter=%d not deterministic: %s != %s",
						in, i, got, ref)
				}
			}
		})
	}
}

// TestTaylorLn1PlusX_Determinism — same guarantee for the log function.
func TestTaylorLn1PlusX_Determinism(t *testing.T) {
	inputs := []string{
		"-0.99",
		"-0.5",
		"-0.001",
		"0",
		"0.001",
		"0.5",
		"1.0",
		"5.0",
	}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			x := sdkmath.LegacyMustNewDecFromStr(in)
			ref := TaylorLn1PlusX(x)
			for i := 0; i < 256; i++ {
				if got := TaylorLn1PlusX(x); !got.Equal(ref) {
					t.Fatalf("TaylorLn1PlusX(%s) iter=%d not deterministic: %s != %s",
						in, i, got, ref)
				}
			}
		})
	}
}

// TestExpApprox_DeterminismAcrossGoroutines — confirms no shared mutable
// state corrupts results when called concurrently (e.g., from EndBlocker
// running across multiple validator goroutines).
func TestExpApprox_DeterminismAcrossGoroutines(t *testing.T) {
	x := sdkmath.LegacyMustNewDecFromStr("0.5")
	ref := ExpApprox(x)

	const workers = 32
	const iters = 100
	var wg sync.WaitGroup
	wg.Add(workers)
	errCh := make(chan string, workers*iters)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				if got := ExpApprox(x); !got.Equal(ref) {
					errCh <- got.String()
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for badStr := range errCh {
		t.Fatalf("concurrent ExpApprox produced %s, expected %s", badStr, ref)
	}
}

// TestSigmoidApprox_Determinism — protects RL consensus reward calculation.
func TestSigmoidApprox_Determinism(t *testing.T) {
	for _, in := range []string{"-3", "-1", "-0.1", "0", "0.1", "1", "3"} {
		x := sdkmath.LegacyMustNewDecFromStr(in)
		ref := SigmoidApprox(x)
		for i := 0; i < 64; i++ {
			if got := SigmoidApprox(x); !got.Equal(ref) {
				t.Fatalf("SigmoidApprox(%s) not deterministic: %s != %s", in, got, ref)
			}
		}
	}
}

// TestReputationMultiplier_BoundsAndDeterminism — covers the consensus-
// safe reputation multiplier used in validator weighting.
func TestReputationMultiplier_BoundsAndDeterminism(t *testing.T) {
	for _, in := range []string{"0", "0.25", "0.5", "0.75", "1.0"} {
		r := sdkmath.LegacyMustNewDecFromStr(in)
		ref := ReputationMultiplier(r)
		for i := 0; i < 64; i++ {
			if got := ReputationMultiplier(r); !got.Equal(ref) {
				t.Fatalf("ReputationMultiplier(%s) not deterministic: %s != %s", in, got, ref)
			}
		}
	}
}
