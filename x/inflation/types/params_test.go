package types

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
)

func dec(s string) math.LegacyDec { return math.LegacyMustNewDecFromStr(s) }

// Params written by a binary that predates emission_mode/cap must keep minting
// exactly as before. This is the upgrade-safety case: on the block after the
// upgrade the module reads old JSON out of the store.
func TestLegacyParamsUnchanged(t *testing.T) {
	var p Params
	legacy := `{"schedule":[{"year":1,"inflation_rate":"0.175000000000000000"}],"epoch_length":43200,"enabled":true}`
	if err := json.Unmarshal([]byte(legacy), &p); err != nil {
		t.Fatalf("legacy params must decode: %v", err)
	}
	if p.Mode() != EmissionModeSchedule {
		t.Fatalf("legacy params must default to schedule mode, got %q", p.Mode())
	}
	if !p.EmissionCap().IsZero() {
		t.Fatalf("legacy params must be uncapped, got %s", p.EmissionCap())
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("legacy params must validate: %v", err)
	}
	// 4.485.743.877 QOR at 17.5%/365 — the live mainnet figure.
	supply := math.NewInt(4_485_743_877_000_000)
	got := p.EpochEmission(dec("0.175"), supply, math.ZeroInt())
	want := dec("0.175").MulInt(supply).QuoInt64(365).TruncateInt()
	if !got.Equal(want) {
		t.Fatalf("schedule emission changed: got %s want %s", got, want)
	}
}

func TestFixedMode(t *testing.T) {
	p := DefaultParams()
	p.EmissionMode = EmissionModeFixed
	p.FixedEpochEmission = OptionalInt(math.NewInt(469_500_000_000))
	if err := p.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	// Supply and rate are irrelevant in fixed mode.
	got := p.EpochEmission(dec("0.175"), math.NewInt(4_485_743_877_000_000), math.ZeroInt())
	if !got.Equal(p.FixedEmission()) {
		t.Fatalf("got %s want %s", got, p.FixedEmission())
	}
	got = p.EpochEmission(math.LegacyZeroDec(), math.ZeroInt(), math.ZeroInt())
	if !got.Equal(p.FixedEmission()) {
		t.Fatalf("fixed mode must not depend on rate or supply, got %s", got)
	}
}

func TestFixedModeRejectsZero(t *testing.T) {
	p := DefaultParams()
	p.EmissionMode = EmissionModeFixed
	if err := p.Validate(); err == nil {
		t.Fatal("fixed mode without an amount must not validate")
	}
}

// The cap must hold exactly: the last epoch is clamped so cumulative minting
// lands on the cap, never above it, and every epoch after that mints nothing.
func TestCapIsExact(t *testing.T) {
	p := DefaultParams()
	p.EmissionMode = EmissionModeFixed
	p.FixedEpochEmission = OptionalInt(math.NewInt(300))
	p.MaxTotalEmission = OptionalInt(math.NewInt(1000))
	if err := p.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	minted := math.ZeroInt()
	for i := 0; i < 10; i++ {
		minted = minted.Add(p.EpochEmission(math.LegacyZeroDec(), math.ZeroInt(), minted))
	}
	if !minted.Equal(math.NewInt(1000)) {
		t.Fatalf("cumulative minting must land exactly on the cap: got %s", minted)
	}
	if e := p.EpochEmission(math.LegacyZeroDec(), math.ZeroInt(), minted); !e.IsZero() {
		t.Fatalf("emission past the cap must be zero, got %s", e)
	}
}

// The same clamp must apply in schedule mode, otherwise a percentage-of-supply
// schedule would blow straight through the ceiling.
func TestCapAppliesInScheduleMode(t *testing.T) {
	p := DefaultParams()
	p.MaxTotalEmission = OptionalInt(math.NewInt(1_000_000))
	supply := math.NewInt(4_485_743_877_000_000)
	got := p.EpochEmission(dec("0.175"), supply, math.ZeroInt())
	if !got.Equal(math.NewInt(1_000_000)) {
		t.Fatalf("first epoch must be clamped to the cap: got %s", got)
	}
	if e := p.EpochEmission(dec("0.175"), supply, math.NewInt(1_000_000)); !e.IsZero() {
		t.Fatalf("emission past the cap must be zero, got %s", e)
	}
}

func TestZeroRateMintsNothing(t *testing.T) {
	p := DefaultParams()
	if e := p.EpochEmission(math.LegacyZeroDec(), math.NewInt(1_000_000), math.ZeroInt()); !e.IsZero() {
		t.Fatalf("zero rate must mint nothing, got %s", e)
	}
}

// A nil math.Int comes back from JSON that never had the key. Reading one must
// not panic — that would halt every validator at the same height.
func TestNilFieldsDoNotPanic(t *testing.T) {
	var p Params
	if err := json.Unmarshal([]byte(`{"schedule":[],"epoch_length":1,"enabled":true}`), &p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	_ = p.FixedEmission()
	_ = p.EmissionCap()
	_ = p.EpochEmission(dec("0.1"), math.NewInt(100), math.Int{})
}

func TestRoundTripKeepsNewFields(t *testing.T) {
	p := DefaultParams()
	p.EmissionMode = EmissionModeFixed
	p.FixedEpochEmission = OptionalInt(math.NewInt(469_500_000_000))
	p.MaxTotalEmission = OptionalInt(math.NewInt(590_000_000_000_000))
	bz, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Params
	if err := json.Unmarshal(bz, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Mode() != EmissionModeFixed || !back.FixedEmission().Equal(p.FixedEmission()) || !back.EmissionCap().Equal(p.EmissionCap()) {
		t.Fatalf("round trip lost data: %+v", back)
	}
}

// The historical formula divided the annual rate by 365 whatever the epoch
// length. The replacement spreads it by block count, which must give the very
// same integer for the mainnet epoch length on every supply, or a node
// replaying history with the new binary mints a different amount and forks.
func TestMainnetEpochLengthKeepsTheHistoricalEmission(t *testing.T) {
	p := DefaultParams()
	p.EpochLength = BlocksPerDay
	for _, supply := range []int64{
		1, 365, 366, 1_000_000, 4_485_743_877_000_000, 4_485_743_877_123_456, 9_999_999_999_999_999,
	} {
		for _, rate := range []string{"0.175", "0.11", "0.07", "0.02", "0.001234567890123456"} {
			got := p.EpochEmission(dec(rate), math.NewInt(supply), math.ZeroInt())
			want := dec(rate).MulInt(math.NewInt(supply)).QuoInt64(365).TruncateInt()
			if !got.Equal(want) {
				t.Fatalf("supply %d rate %s: got %s want %s", supply, rate, got, want)
			}
		}
	}
	if p.EpochsPerYear() != 365 {
		t.Fatalf("mainnet epoch length must give 365 epochs per year, got %d", p.EpochsPerYear())
	}
}

// A 30-block epoch is 1/1440 of a day, so it mints 1/1440 of a day's emission,
// not a whole day's. This is the public testnet's configuration; at the old
// formula it minted 69.9 million QOR every 30 seconds.
func TestShortEpochMintsItsShareOfTheYear(t *testing.T) {
	p := DefaultParams()
	p.EpochLength = 30
	supply := math.NewInt(1_400_000_000_000_000)
	got := p.EpochEmission(dec("0.02"), supply, math.ZeroInt())
	day := dec("0.02").MulInt(supply).QuoInt64(365).TruncateInt()
	if !got.Mul(math.NewInt(1440)).LTE(day) || got.Mul(math.NewInt(1440)).LT(day.Sub(math.NewInt(1440))) {
		t.Fatalf("30-block epoch must mint one 1440th of a day: got %s, day %s", got, day)
	}
	if p.EpochsPerYear() != 525_600 {
		t.Fatalf("30-block epochs: want 525,600 per year, got %d", p.EpochsPerYear())
	}
}

// The two formulas, side by side, on the testnet's 30-block epochs: the
// historical one mints a full day per epoch (what every block below the
// upgrade height was made with, and must replay byte-identically), the scaled
// one mints the 30-block share of the year. On the mainnet epoch length the
// two agree to the integer.
func TestEpochEmissionScalingIsASwitch(t *testing.T) {
	rate := math.LegacyNewDecWithPrec(2, 2)      // 2%
	supply := math.NewInt(6_702_460_660_317_650) // uqor
	p := DefaultParams()
	p.EmissionMode = EmissionModeSchedule
	p.MaxTotalEmission = nil

	p.EpochLength = 30
	legacy := p.EpochEmissionScaled(rate, supply, math.ZeroInt(), false)
	scaled := p.EpochEmissionScaled(rate, supply, math.ZeroInt(), true)
	wantLegacy := rate.MulInt(supply).QuoInt64(365).TruncateInt()
	if !legacy.Equal(wantLegacy) {
		t.Fatalf("legacy formula changed: got %s want %s", legacy, wantLegacy)
	}
	if !scaled.Mul(math.NewInt(1440)).LTE(legacy) || scaled.Mul(math.NewInt(1441)).LT(legacy) {
		t.Fatalf("scaled emission is not the 30-block share of a day: legacy %s scaled %s", legacy, scaled)
	}
	if !p.EpochEmission(rate, supply, math.ZeroInt()).Equal(scaled) {
		t.Fatal("EpochEmission must be the scaled form")
	}

	p.EpochLength = BlocksPerDay
	if a, b := p.EpochEmissionScaled(rate, supply, math.ZeroInt(), false), p.EpochEmissionScaled(rate, supply, math.ZeroInt(), true); !a.Equal(b) {
		t.Fatalf("on the mainnet epoch length the two formulas must agree: %s vs %s", a, b)
	}
}
