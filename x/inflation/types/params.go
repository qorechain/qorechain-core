package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// EmissionTier defines the inflation rate for a given year.
type EmissionTier struct {
	Year          uint64         `json:"year"`
	InflationRate math.LegacyDec `json:"inflation_rate"`
}

// Params defines the configurable parameters for the inflation module.
type Params struct {
	Schedule    []EmissionTier `json:"schedule"`
	EpochLength int64          `json:"epoch_length"` // blocks per epoch
	Enabled     bool           `json:"enabled"`
}

// DefaultParams returns the default inflation parameters.
func DefaultParams() Params {
	return Params{
		Schedule: []EmissionTier{
			{Year: 1, InflationRate: math.LegacyNewDecWithPrec(175, 3)}, // 17.5%
			{Year: 2, InflationRate: math.LegacyNewDecWithPrec(11, 2)},  // 11%
			{Year: 3, InflationRate: math.LegacyNewDecWithPrec(7, 2)},   // 7%
			{Year: 4, InflationRate: math.LegacyNewDecWithPrec(7, 2)},   // 7%
			{Year: 5, InflationRate: math.LegacyNewDecWithPrec(2, 2)},   // 2% (perpetual)
		},
		EpochLength: 17280, // ~1 day at 5s blocks
		Enabled:     true,
	}
}

// Validate checks param correctness.
func (p Params) Validate() error {
	if len(p.Schedule) == 0 {
		return fmt.Errorf("emission schedule must not be empty")
	}
	for i, tier := range p.Schedule {
		if tier.InflationRate.IsNegative() {
			return fmt.Errorf("schedule[%d]: inflation_rate must be non-negative", i)
		}
		if tier.InflationRate.GT(math.LegacyOneDec()) {
			return fmt.Errorf("schedule[%d]: inflation_rate must be <= 1.0", i)
		}
	}
	// Verify years are strictly increasing
	for i := 1; i < len(p.Schedule); i++ {
		if p.Schedule[i].Year <= p.Schedule[i-1].Year {
			return fmt.Errorf("schedule[%d]: year must be strictly increasing", i)
		}
	}
	if p.EpochLength < 1 {
		return fmt.Errorf("epoch_length must be >= 1")
	}
	return nil
}
