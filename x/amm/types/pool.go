package types

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
)

// PoolType selects the pricing curve.
type PoolType string

const (
	// PoolTypeConstantProduct implements x*y=k (Uniswap-V2 style).
	PoolTypeConstantProduct PoolType = "constant_product"

	// PoolTypeStableSwap implements Curve-style invariant for low-slippage
	// stable pairs. Solved via Newton iteration with a fixed step cap.
	PoolTypeStableSwap PoolType = "stable_swap"
)

// PoolStatus tracks lifecycle state.
type PoolStatus string

const (
	PoolStatusActive PoolStatus = "active"
	PoolStatusPaused PoolStatus = "paused"
)

// Pool is a single liquidity pool.
//
// Reserves are stored as math.Int (uint256-equivalent unsigned integer)
// to avoid floating-point arithmetic in consensus paths. The lp_supply
// is the total LP token supply minted against this pool — burning LP
// tokens redeems a proportional share of (ReserveA, ReserveB).
//
// WeightedAvgPrice is a TWAP-like running average of (ReserveB / ReserveA),
// recomputed in EndBlocker. Stored as a sdkmath.LegacyDec scaled by 10^18.
type Pool struct {
	ID               uint64         `json:"id"`
	Type             PoolType       `json:"type"`
	Creator          string         `json:"creator"` // bech32
	TokenA           string         `json:"token_a"` // sorted lexicographically: TokenA < TokenB
	TokenB           string         `json:"token_b"`
	ReserveA         math.Int       `json:"reserve_a"`
	ReserveB         math.Int       `json:"reserve_b"`
	LPSupply         math.Int       `json:"lp_supply"`
	LPDenom          string         `json:"lp_denom"`
	CreatedAt        int64          `json:"created_at"` // block height at creation
	Status           PoolStatus     `json:"status"`
	WeightedAvgPrice math.LegacyDec `json:"weighted_avg_price"`

	// StableSwap-specific. Ignored for ConstantProduct.
	// AmplificationCoefficient (A): higher A → lower slippage near equilibrium.
	// Curve docs use 100–500 for stable pairs. Bounded [1, 5000] in Validate.
	AmplificationCoefficient uint32 `json:"amplification_coefficient,omitempty"`
}

// LPDenomFor returns the canonical LP denom for a given pool ID.
func LPDenomFor(poolID uint64) string {
	return fmt.Sprintf("%s/%d", LPDenomPrefix, poolID)
}

// SortedDenomTuple returns (a, b) such that a < b lexicographically. The AMM
// stores pools with sorted denoms so that lookups by (X, Y) and (Y, X)
// resolve to the same pool.
func SortedDenomTuple(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// Validate checks invariants on a Pool.
func (p Pool) Validate() error {
	if p.ID == 0 {
		return fmt.Errorf("pool id must be > 0")
	}
	if p.Type != PoolTypeConstantProduct && p.Type != PoolTypeStableSwap {
		return fmt.Errorf("invalid pool type %q", p.Type)
	}
	if strings.TrimSpace(p.Creator) == "" {
		return fmt.Errorf("pool creator must be set")
	}
	if p.TokenA == p.TokenB {
		return ErrSameDenom
	}
	if p.TokenA == "" || p.TokenB == "" {
		return ErrInvalidDenoms
	}
	if p.TokenA >= p.TokenB {
		return fmt.Errorf("denoms must be sorted lexicographically (TokenA < TokenB)")
	}
	if p.ReserveA.IsNil() || p.ReserveB.IsNil() {
		return fmt.Errorf("reserves must not be nil")
	}
	if p.ReserveA.IsNegative() || p.ReserveB.IsNegative() {
		return fmt.Errorf("reserves must be non-negative")
	}
	if p.LPSupply.IsNil() || p.LPSupply.IsNegative() {
		return fmt.Errorf("lp_supply must be non-negative")
	}
	// LP supply must be zero iff both reserves are zero (drained pool).
	bothEmpty := p.ReserveA.IsZero() && p.ReserveB.IsZero()
	if bothEmpty != p.LPSupply.IsZero() {
		return fmt.Errorf("lp_supply / reserves consistency violated: empty=%v lp_zero=%v", bothEmpty, p.LPSupply.IsZero())
	}
	if p.LPDenom != LPDenomFor(p.ID) {
		return fmt.Errorf("lp_denom %q does not match expected %q", p.LPDenom, LPDenomFor(p.ID))
	}
	if p.Status != PoolStatusActive && p.Status != PoolStatusPaused {
		return fmt.Errorf("invalid pool status %q", p.Status)
	}
	if p.WeightedAvgPrice.IsNil() {
		return fmt.Errorf("weighted_avg_price must not be nil")
	}
	if p.WeightedAvgPrice.IsNegative() {
		return fmt.Errorf("weighted_avg_price must be non-negative")
	}
	if p.Type == PoolTypeStableSwap {
		if p.AmplificationCoefficient < 1 || p.AmplificationCoefficient > 5000 {
			return fmt.Errorf("amplification_coefficient must be in [1,5000] for stable_swap, got %d", p.AmplificationCoefficient)
		}
	}
	return nil
}
