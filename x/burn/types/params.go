package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Params defines the configurable parameters for the burn module.
type Params struct {
	GasBurnRate       math.LegacyDec `json:"gas_burn_rate"`        // 0.30 — 30% of fees burned
	ContractCreateFee math.Int       `json:"contract_create_fee"`  // flat QOR fee for contract creation
	AIServiceBurnRate math.LegacyDec `json:"ai_service_burn_rate"` // 0.50 — 50% of AI service fees
	BridgeBurnRate    math.LegacyDec `json:"bridge_burn_rate"`     // 1.00 — 100% of bridge fees
	FailedTxBurnRate  math.LegacyDec `json:"failed_tx_burn_rate"`  // partial gas burn on failure
	ValidatorShare    math.LegacyDec `json:"validator_share"`      // 0.40 — 40% to validators
	TreasuryShare     math.LegacyDec `json:"treasury_share"`       // 0.20 — 20% to treasury
	StakerShare       math.LegacyDec `json:"staker_share"`         // 0.10 — 10% to stakers
	Enabled           bool           `json:"enabled"`
}

// DefaultParams returns the default burn module parameters.
func DefaultParams() Params {
	return Params{
		GasBurnRate:       math.LegacyNewDecWithPrec(30, 2), // 0.30
		ContractCreateFee: math.NewInt(100_000_000),          // 100 QOR in uqor
		AIServiceBurnRate: math.LegacyNewDecWithPrec(50, 2), // 0.50
		BridgeBurnRate:    math.LegacyOneDec(),               // 1.00
		FailedTxBurnRate:  math.LegacyNewDecWithPrec(10, 2), // 0.10
		ValidatorShare:    math.LegacyNewDecWithPrec(40, 2), // 0.40
		TreasuryShare:     math.LegacyNewDecWithPrec(20, 2), // 0.20
		StakerShare:       math.LegacyNewDecWithPrec(10, 2), // 0.10
		Enabled:           true,
	}
}

// Validate checks param correctness.
func (p Params) Validate() error {
	if p.GasBurnRate.IsNegative() || p.GasBurnRate.GT(math.LegacyOneDec()) {
		return fmt.Errorf("gas_burn_rate must be between 0 and 1, got %s", p.GasBurnRate)
	}
	if p.ContractCreateFee.IsNegative() {
		return fmt.Errorf("contract_create_fee must be non-negative")
	}
	if p.AIServiceBurnRate.IsNegative() || p.AIServiceBurnRate.GT(math.LegacyOneDec()) {
		return fmt.Errorf("ai_service_burn_rate must be between 0 and 1")
	}
	if p.BridgeBurnRate.IsNegative() || p.BridgeBurnRate.GT(math.LegacyOneDec()) {
		return fmt.Errorf("bridge_burn_rate must be between 0 and 1")
	}
	if p.FailedTxBurnRate.IsNegative() || p.FailedTxBurnRate.GT(math.LegacyOneDec()) {
		return fmt.Errorf("failed_tx_burn_rate must be between 0 and 1")
	}
	// validator_share + burn (gas_burn_rate) + treasury + staker must equal 1.0
	totalShares := p.ValidatorShare.Add(p.GasBurnRate).Add(p.TreasuryShare).Add(p.StakerShare)
	if !totalShares.Equal(math.LegacyOneDec()) {
		return fmt.Errorf("fee shares must sum to 1.0, got %s", totalShares)
	}
	return nil
}
