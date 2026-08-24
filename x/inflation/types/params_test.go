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
