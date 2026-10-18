package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Emission modes.
//
// EmissionModeSchedule is the original behaviour: each epoch mints
// totalSupply * rate * epoch_length / blocks_per_year (rate / 365 at the
// mainnet epoch length). It is self-feeding — supply grows, so the base
// grows, so the next epoch mints more. It can never bound a total.
//
// EmissionModeFixed mints a constant amount per epoch, which together with
// MaxTotalEmission turns an allocation into an actual ceiling.
//
// EmissionModeTargetAPY (v3.2.4, protocol feature inflation-target-apy) mints
// per epoch what the bonded stake needs to earn TargetAPYBps a year through the
// fee split, still clamped to MaxTotalEmission. See TargetAPYEmission.
const (
	EmissionModeSchedule  = "schedule"
	EmissionModeFixed     = "fixed"
	EmissionModeTargetAPY = "target_apy"
)

// SecondsPerYear is the year a target yield is stated against: 365 days.
const SecondsPerYear = int64(365 * 24 * 60 * 60)

// BasisPoints is 100%.
const BasisPoints = int64(10_000)

// BlocksPerDay and BlocksPerYear fix the calendar the schedule is written
// against: an annual rate, one day per 43,200 blocks (2 s target). Epoch
// lengths are converted to a share of the year through these constants, so a
// chain that chooses shorter epochs mints proportionally less per epoch and
// advances the schedule year after the same number of blocks.
const (
	BlocksPerDay  = int64(43_200)
	BlocksPerYear = 365 * BlocksPerDay
)

// EpochsPerYear is how many epochs of this length make up a schedule year,
// never less than one. 365 for the mainnet epoch length.
func (p Params) EpochsPerYear() uint64 {
	if p.EpochLength < 1 {
		return 365
	}
	n := BlocksPerYear / p.EpochLength
	if n < 1 {
		return 1
	}
	return uint64(n)
}

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

	// TargetAPYBps is the yearly yield, in basis points of the bonded stake,
	// that EmissionModeTargetAPY aims to pay stakers before commission. It is
	// a gross figure: a delegator nets it less their validator's commission,
	// so 1667 is about 15% net at a 10% commission (1667 x 0.90 = 1500).
	//
	// Zero is omitted from the stored JSON, so params written before this
	// field existed keep their bytes.
	TargetAPYBps uint64 `json:"target_apy_bps,omitempty"`
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
	case EmissionModeTargetAPY:
		// The mode is accepted only with a target it can act on. The emission
		// cap still applies; the target only says how fast to approach it.
		if p.TargetAPYBps == 0 {
			return fmt.Errorf("target_apy_bps must be positive in %q mode", EmissionModeTargetAPY)
		}
	default:
		return fmt.Errorf("emission_mode must be %q, %q or %q, got %q",
			EmissionModeSchedule, EmissionModeFixed, EmissionModeTargetAPY, p.EmissionMode)
	}
	if p.TargetAPYBps > uint64(BasisPoints) {
		return fmt.Errorf("target_apy_bps must be at most %d (100%%), got %d", BasisPoints, p.TargetAPYBps)
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
	return p.EpochEmissionScaled(rate, totalSupply, alreadyMinted, true)
}

// EpochEmissionScaled is EpochEmission with the epoch-length scaling as a
// switch. Off, it is the historical formula (a day's emission per epoch,
// whatever the epoch length), which every block below the release upgrade
// height was made with and must replay byte-identically; on, the corrected
// one. The switch is a protocol feature the upgrade handler turns on.
func (p Params) EpochEmissionScaled(rate math.LegacyDec, totalSupply, alreadyMinted math.Int, scaled bool) math.Int {
	var amount math.Int
	switch p.Mode() {
	case EmissionModeFixed:
		amount = p.FixedEmission()
	default:
		if rate.IsNil() || rate.IsZero() || totalSupply.IsNil() || !totalSupply.IsPositive() {
			return math.ZeroInt()
		}
		if !scaled {
			const epochsPerYear = int64(365)
			amount = rate.MulInt(totalSupply).QuoInt64(epochsPerYear).TruncateInt()
			break
		}
		// The annual rate is spread over the year by block count, so an epoch
		// mints the share of the year its length represents. For the mainnet
		// convention (43,200 blocks per epoch, one day at 2 s) this is exactly
		// rate * supply / 365, the historical formula, and it is computed on the
		// same rational so the truncated integer is byte-identical on replay.
		// Any other epoch length used to mint a full day's emission regardless:
		// the public testnet, configured with 30-block epochs, minted a day's
		// worth every 30 seconds and grew its supply 900-fold.
		amount = rate.MulInt(totalSupply).MulInt64(p.EpochLength).QuoInt64(BlocksPerYear).TruncateInt()
	}
	return p.ClampToCap(amount, alreadyMinted)
}

// ClampToCap returns amount, reduced so that alreadyMinted plus the result
// never exceeds MaxTotalEmission. Zero when the cap is reached or amount is
// not positive; amount unchanged when there is no cap.
func (p Params) ClampToCap(amount, alreadyMinted math.Int) math.Int {
	if amount.IsNil() || !amount.IsPositive() {
		return math.ZeroInt()
	}
	remaining, capped := p.RemainingCap(alreadyMinted)
	if !capped {
		return amount
	}
	if amount.GT(remaining) {
		return remaining
	}
	return amount
}

// RemainingCap reports how much may still be minted under MaxTotalEmission
// after alreadyMinted, floored at zero. capped is false when there is no cap,
// and the amount is then meaningless.
func (p Params) RemainingCap(alreadyMinted math.Int) (remaining math.Int, capped bool) {
	cap := p.EmissionCap()
	if !cap.IsPositive() {
		return math.ZeroInt(), false
	}
	minted := alreadyMinted
	if minted.IsNil() {
		minted = math.ZeroInt()
	}
	remaining = cap.Sub(minted)
	if !remaining.IsPositive() {
		return math.ZeroInt(), true
	}
	return remaining, true
}

// NominalEpochSeconds is how long an epoch lasts on the schedule's calendar
// (BlocksPerDay blocks a day): 86,400 s for the mainnet epoch length.
func (p Params) NominalEpochSeconds() int64 {
	if p.EpochLength < 1 {
		return 0
	}
	return p.EpochLength * 86_400 / BlocksPerDay
}

// TargetAPYEmission is the target_apy mode's emission for one epoch of
// epochSeconds, before the cap:
//
//	bonded x target_apy_bps / 10,000 x epochSeconds / SecondsPerYear / stakerReach
//
// stakerReach is the share of every minted coin that reaches stakers through
// the fee split (x/burn validator_share + staker_share, 0.37 + 0.10 = 0.47
// today). Minting bonded x rate / reach makes reach x minted, the part that
// actually reaches the bonded stake, equal to bonded x rate.
//
// Worked through for the default target of 1667 bps and a reach of 0.47: a
// year mints 0.1667 / 0.47 = 35.47% of the bonded stake, of which 47% (16.67%
// of the bonded stake) reaches stakers; after a 10% commission a delegator
// nets 16.67% x 0.90 = 15.0%. The cumulative cap is applied by the caller.
//
// Integer and LegacyDec arithmetic only; zero for any non-positive input.
func TargetAPYEmission(bonded math.Int, targetBps uint64, stakerReach math.LegacyDec, epochSeconds int64) math.Int {
	if bonded.IsNil() || !bonded.IsPositive() || targetBps == 0 || epochSeconds <= 0 ||
		stakerReach.IsNil() || !stakerReach.IsPositive() {
		return math.ZeroInt()
	}
	return math.LegacyNewDecFromInt(bonded).
		MulInt64(int64(targetBps)).
		MulInt64(epochSeconds).
		QuoInt64(BasisPoints).
		QuoInt64(SecondsPerYear).
		Quo(stakerReach).
		TruncateInt()
}
