package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Emission modes.
//
// EmissionModeSchedule is the original behaviour: each epoch mints
// totalSupply * rate / 365. It is self-feeding — supply grows, so the base
// grows, so the next epoch mints more. It can never bound a total.
//
// EmissionModeFixed mints a constant amount per epoch, which together with
// MaxTotalEmission turns an allocation into an actual ceiling.
const (
	EmissionModeSchedule = "schedule"
	EmissionModeFixed    = "fixed"
)

// EmissionTier defines the inflation rate for a given year.
type EmissionTier struct {
	Year          uint64         `json:"year"`
	InflationRate math.LegacyDec `json:"inflation_rate"`
}

// Params defines the configurable parameters for the inflation module.
//
// Params are stored as JSON, so fields added here are backward compatible:
// state written by an older binary unmarshals cleanly and the new fields take
// their zero value. Use the accessors below rather than the raw fields — a
// math.Int decoded from JSON that lacked the key is nil, and calling a method
// on it panics.
type Params struct {
	Schedule    []EmissionTier `json:"schedule"`
	EpochLength int64          `json:"epoch_length"` // blocks per epoch
	Enabled     bool           `json:"enabled"`

	// EmissionMode selects how the per-epoch amount is computed. Empty means
	// EmissionModeSchedule, so pre-existing state keeps its behaviour.
	EmissionMode string `json:"emission_mode,omitempty"`

	// FixedEpochEmission is the uqor minted per epoch when EmissionMode is
	// EmissionModeFixed.
	//
	// POINTER, deliberately. `omitempty` does not work on math.Int: it is a
	// struct, so encoding/json never considers it empty and always writes
	// `"fixed_epoch_emission":"0"`. Params are stored as JSON and SetParams is
	// called by InitGenesis, so a value type here means a node syncing from
	// genesis writes different bytes at height 0 than the running chain has —
	// it could not join. A nil pointer is omitted, so an unset field serialises
	// byte-identically to a record written before this field existed.
	FixedEpochEmission *math.Int `json:"fixed_epoch_emission,omitempty"`

	// MaxTotalEmission caps cumulative minting by this module. Once
	// EpochInfo.TotalMinted reaches it, emission stops permanently. Zero or
	// unset means uncapped. This is what makes an allocation a ceiling rather
	// than a target: it holds regardless of block-time drift, which shifts how
	// many epochs fit in a year.
	//
	// Pointer for the same reason as above.
	MaxTotalEmission *math.Int `json:"max_total_emission,omitempty"`
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
		EpochLength:  17280, // ~1 day at 5s blocks
		Enabled:      true,
		EmissionMode: EmissionModeSchedule,
	}
}

// Mode returns the effective emission mode, treating empty as the legacy
// schedule mode so that params written before this field existed behave
// exactly as they did.
func (p Params) Mode() string {
	if p.EmissionMode == "" {
		return EmissionModeSchedule
	}
	return p.EmissionMode
}

// FixedEmission returns FixedEpochEmission, or zero if it was never set.
func (p Params) FixedEmission() math.Int {
	if p.FixedEpochEmission == nil || p.FixedEpochEmission.IsNil() {
		return math.ZeroInt()
	}
	return *p.FixedEpochEmission
}

// EmissionCap returns MaxTotalEmission, or zero (meaning uncapped) if unset.
func (p Params) EmissionCap() math.Int {
	if p.MaxTotalEmission == nil || p.MaxTotalEmission.IsNil() {
		return math.ZeroInt()
	}
	return *p.MaxTotalEmission
}

// OptionalInt returns a pointer to v, or nil when v carries no information.
// Storing a pointer to zero would defeat the whole point of the pointer: it
// would write the field and change the record's bytes.
func OptionalInt(v math.Int) *math.Int {
	if v.IsNil() || !v.IsPositive() {
		return nil
	}
	out := v
	return &out
}

// Validate checks param correctness.
func (p Params) Validate() error {
	switch p.Mode() {
	case EmissionModeSchedule:
		if len(p.Schedule) == 0 {
			return fmt.Errorf("emission schedule must not be empty in %q mode", EmissionModeSchedule)
		}
	case EmissionModeFixed:
		if !p.FixedEmission().IsPositive() {
			return fmt.Errorf("fixed_epoch_emission must be positive in %q mode", EmissionModeFixed)
		}
	default:
		return fmt.Errorf("emission_mode must be %q or %q, got %q", EmissionModeSchedule, EmissionModeFixed, p.EmissionMode)
	}

	for i, tier := range p.Schedule {
		if tier.InflationRate.IsNil() {
			return fmt.Errorf("schedule[%d]: inflation_rate must be set", i)
		}
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
	if p.EmissionCap().IsNegative() {
		return fmt.Errorf("max_total_emission must be non-negative")
	}
	return nil
}

// EpochEmission returns how much this epoch should mint given the current
// supply, the tier rate and how much has already been minted. It returns zero
// when the cap is reached, and clamps the final epoch so cumulative minting
// lands exactly on the cap rather than overshooting it.
func (p Params) EpochEmission(rate math.LegacyDec, totalSupply, alreadyMinted math.Int) math.Int {
	var amount math.Int
	switch p.Mode() {
	case EmissionModeFixed:
		amount = p.FixedEmission()
	default:
		if rate.IsNil() || rate.IsZero() || totalSupply.IsNil() || !totalSupply.IsPositive() {
			return math.ZeroInt()
		}
		const epochsPerYear = int64(365)
		amount = rate.MulInt(totalSupply).QuoInt64(epochsPerYear).TruncateInt()
	}
	if !amount.IsPositive() {
		return math.ZeroInt()
	}

	cap := p.EmissionCap()
	if !cap.IsPositive() {
		return amount // uncapped
	}
	minted := alreadyMinted
	if minted.IsNil() {
		minted = math.ZeroInt()
	}
	remaining := cap.Sub(minted)
	if !remaining.IsPositive() {
		return math.ZeroInt()
	}
	if amount.GT(remaining) {
		return remaining
	}
	return amount
}
