package types

import (
	"testing"

	"cosmossdk.io/math"
)

func validPool() Pool {
	return Pool{
		ID:               1,
		Type:             PoolTypeConstantProduct,
		Creator:          "qor1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		TokenA:           "uatom",
		TokenB:           "uusdc",
		ReserveA:         math.NewInt(1_000_000),
		ReserveB:         math.NewInt(2_000_000),
		LPSupply:         math.NewInt(1_414_213),
		LPDenom:          LPDenomFor(1),
		CreatedAt:        100,
		Status:           PoolStatusActive,
		WeightedAvgPrice: math.LegacyZeroDec(),
	}
}

func TestPool_Validate_OK(t *testing.T) {
	p := validPool()
	if err := p.Validate(); err != nil {
		t.Fatalf("expected validate ok, got: %v", err)
	}
}

func TestPool_Validate_RejectsSameDenom(t *testing.T) {
	p := validPool()
	p.TokenA = "uatom"
	p.TokenB = "uatom"
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for same denom")
	}
}

func TestPool_Validate_RejectsUnsortedDenoms(t *testing.T) {
	p := validPool()
	p.TokenA = "uusdc" // > "uatom"
	p.TokenB = "uatom"
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for unsorted denoms")
	}
}

func TestPool_Validate_RejectsLPSupplyMismatch(t *testing.T) {
	p := validPool()
	p.ReserveA = math.ZeroInt()
	p.ReserveB = math.ZeroInt()
	// LPSupply non-zero with empty reserves should fail.
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for LP/reserve consistency violation")
	}
}

func TestPool_Validate_RejectsLPDenomMismatch(t *testing.T) {
	p := validPool()
	p.LPDenom = "wrong-prefix/1"
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for wrong LP denom")
	}
}

func TestPool_Validate_RejectsStableSwapBadAmplification(t *testing.T) {
	p := validPool()
	p.Type = PoolTypeStableSwap
	p.AmplificationCoefficient = 0
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for amplification_coefficient = 0 with StableSwap")
	}
	p.AmplificationCoefficient = 10000
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for amplification_coefficient > 5000")
	}
}

func TestSortedDenomTuple(t *testing.T) {
	cases := []struct {
		a, b, wantA, wantB string
	}{
		{"uatom", "uusdc", "uatom", "uusdc"},
		{"uusdc", "uatom", "uatom", "uusdc"},
		{"a", "a", "a", "a"},
	}
	for _, c := range cases {
		gotA, gotB := SortedDenomTuple(c.a, c.b)
		if gotA != c.wantA || gotB != c.wantB {
			t.Errorf("SortedDenomTuple(%q,%q) = (%q,%q), want (%q,%q)", c.a, c.b, gotA, gotB, c.wantA, c.wantB)
		}
	}
}

func TestLPDenomFor_Format(t *testing.T) {
	if got := LPDenomFor(42); got != "amm-lp/42" {
		t.Errorf("LPDenomFor(42) = %q, want amm-lp/42", got)
	}
}
