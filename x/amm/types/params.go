package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Params defines the configurable parameters for the AMM module.
type Params struct {
	// SwapFeeBps is the fee charged on each swap, in basis points (1bps = 0.01%).
	// Default 30 = 0.30%.
	SwapFeeBps uint32 `json:"swap_fee_bps"`

	// ProtocolFeeBps is the portion of the swap fee that is forwarded to the
	// burn/treasury via x/burn (BurnSourceAMM). Must be <= SwapFeeBps.
	// The remaining (SwapFeeBps - ProtocolFeeBps) accrues to liquidity providers.
	// Default 10 = 0.10%.
	ProtocolFeeBps uint32 `json:"protocol_fee_bps"`

	// MinLiquidity is the minimum reserve value required to create a pool,
	// expressed in uqor-equivalent (denom A) units. Default 1000.
	MinLiquidity math.Int `json:"min_liquidity"`

	// MaxPoolsPerCreator caps how many distinct pools a single address may create.
	// Default 50. Zero means no cap (governance only).
	MaxPoolsPerCreator uint32 `json:"max_pools_per_creator"`

	// LPTokenDecimals is the cosmetic decimals for LP token denoms. Default 6.
	LPTokenDecimals uint32 `json:"lp_token_decimals"`

	// PoolCreationFee is the one-shot fee charged at MsgCreatePool time, in uqor.
	// Burned via BurnSourceAMM. Default 100_000_000 (100 QOR).
	PoolCreationFee math.Int `json:"pool_creation_fee"`

	// MaxSwapImpactBps caps the slippage allowed in a single swap relative to the
	// pre-swap mid-price, in basis points. Default 500 = 5.00%. Zero disables.
	MaxSwapImpactBps uint32 `json:"max_swap_impact_bps"`

	// Enabled is the module-wide kill switch. When false, all state-mutating
	// messages return ErrPoolPaused. Read paths still work.
	Enabled bool `json:"enabled"`
}

// DefaultParams returns the v3.0.0 default AMM parameters.
func DefaultParams() Params {
	return Params{
		SwapFeeBps:         30,
		ProtocolFeeBps:     10,
		MinLiquidity:       math.NewInt(1000),
		MaxPoolsPerCreator: 50,
		LPTokenDecimals:    6,
		PoolCreationFee:    math.NewInt(100_000_000),
		MaxSwapImpactBps:   500,
		Enabled:            true,
	}
}

// MaxFeeBps caps the absolute fee at 10% (1000 bps) — any value above this is
// rejected by Validate to protect users from misconfigured pools.
const MaxFeeBps uint32 = 1000

// Validate checks that the params are internally consistent.
func (p Params) Validate() error {
	if p.SwapFeeBps > MaxFeeBps {
		return fmt.Errorf("swap_fee_bps must be <= %d, got %d", MaxFeeBps, p.SwapFeeBps)
	}
	if p.ProtocolFeeBps > p.SwapFeeBps {
		return fmt.Errorf("protocol_fee_bps (%d) must be <= swap_fee_bps (%d)", p.ProtocolFeeBps, p.SwapFeeBps)
	}
	if p.MinLiquidity.IsNil() || !p.MinLiquidity.IsPositive() {
		return fmt.Errorf("min_liquidity must be positive, got %v", p.MinLiquidity)
	}
	if p.LPTokenDecimals == 0 || p.LPTokenDecimals > 18 {
		return fmt.Errorf("lp_token_decimals must be in (0, 18], got %d", p.LPTokenDecimals)
	}
	if p.PoolCreationFee.IsNil() || p.PoolCreationFee.IsNegative() {
		return fmt.Errorf("pool_creation_fee must be non-negative, got %v", p.PoolCreationFee)
	}
	if p.MaxSwapImpactBps > 10000 {
		return fmt.Errorf("max_swap_impact_bps must be <= 10000, got %d", p.MaxSwapImpactBps)
	}
	return nil
}
