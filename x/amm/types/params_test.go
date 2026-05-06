package types

import (
	"testing"

	"cosmossdk.io/math"
)

func TestDefaultParams_Validate(t *testing.T) {
	if err := DefaultParams().Validate(); err != nil {
		t.Fatalf("DefaultParams should validate, got: %v", err)
	}
}

func TestDefaultParams_LocksV3Defaults(t *testing.T) {
	p := DefaultParams()
	cases := []struct {
		name string
		got  uint32
		want uint32
	}{
		{"swap_fee_bps", p.SwapFeeBps, 30},
		{"protocol_fee_bps", p.ProtocolFeeBps, 10},
		{"lp_token_decimals", p.LPTokenDecimals, 6},
		{"max_pools_per_creator", p.MaxPoolsPerCreator, 50},
		{"max_swap_impact_bps", p.MaxSwapImpactBps, 500},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, c.got, c.want)
		}
	}
	if !p.MinLiquidity.Equal(math.NewInt(1000)) {
		t.Errorf("min_liquidity: got %s, want 1000", p.MinLiquidity)
	}
	if !p.PoolCreationFee.Equal(math.NewInt(100_000_000)) {
		t.Errorf("pool_creation_fee: got %s, want 100_000_000", p.PoolCreationFee)
	}
	if !p.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestParams_Validate_RejectsExcessSwapFee(t *testing.T) {
	p := DefaultParams()
	p.SwapFeeBps = MaxFeeBps + 1
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for fee > MaxFeeBps")
	}
}

func TestParams_Validate_RejectsProtocolFeeOverSwapFee(t *testing.T) {
	p := DefaultParams()
	p.ProtocolFeeBps = p.SwapFeeBps + 1
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for protocol_fee > swap_fee")
	}
}

func TestParams_Validate_RejectsZeroLPDecimals(t *testing.T) {
	p := DefaultParams()
	p.LPTokenDecimals = 0
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for lp_token_decimals = 0")
	}
}

func TestParams_Validate_RejectsLPDecimalsOver18(t *testing.T) {
	p := DefaultParams()
	p.LPTokenDecimals = 19
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for lp_token_decimals > 18")
	}
}

func TestParams_Validate_RejectsNonPositiveMinLiquidity(t *testing.T) {
	p := DefaultParams()
	p.MinLiquidity = math.ZeroInt()
	if err := p.Validate(); err == nil {
		t.Fatal("expected error for zero min_liquidity")
	}
}
