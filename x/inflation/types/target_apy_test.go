package types

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
)

func targetParams(bps uint64) Params {
	p := DefaultParams()
	p.EmissionMode = EmissionModeTargetAPY
	p.TargetAPYBps = bps
	return p
}

// The mode is accepted only with a target it can act on.
func TestTargetAPYModeValidation(t *testing.T) {
	if err := targetParams(1667).Validate(); err != nil {
		t.Fatalf("target_apy with 1667 bps must validate: %v", err)
	}
	if err := targetParams(0).Validate(); err == nil {
		t.Fatal("target_apy without a target must be refused")
	}
	if err := targetParams(10_001).Validate(); err == nil {
		t.Fatal("a target above 100% must be refused")
	}
	if err := targetParams(10_000).Validate(); err != nil {
		t.Fatalf("a target of exactly 100%% is a valid bound: %v", err)
	}
	// Another mode with an out-of-range target left over is refused too, so a
	// later switch to target_apy cannot inherit a nonsensical value.
	p := DefaultParams()
	p.TargetAPYBps = 20_000
	if err := p.Validate(); err == nil {
		t.Fatal("an out-of-range target must be refused in any mode")
	}
}

// The message carries the target into the params.
func TestMsgUpdateParamsCarriesTarget(t *testing.T) {
	m := MsgUpdateParams{
		Schedule:     []Tier{{Year: 1, InflationRate: dec("0.02")}},
		EpochLength:  43_200,
		Enabled:      true,
		EmissionMode: EmissionModeTargetAPY,
		TargetApyBps: 1667,
	}
	p := m.ToParams()
	if p.Mode() != EmissionModeTargetAPY || p.TargetAPYBps != 1667 {
		t.Fatalf("got mode %q target %d", p.Mode(), p.TargetAPYBps)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

// A zero target is omitted from the stored JSON: params written before the
// field existed keep their bytes, and a set target round-trips.
func TestTargetAPYBpsBytes(t *testing.T) {
	bz, err := json.Marshal(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(bz, &probe); err != nil {
		t.Fatal(err)
	}
	if _, ok := probe["target_apy_bps"]; ok {
		t.Fatalf("an unset target must not be written: %s", bz)
	}
	bz, err = json.Marshal(targetParams(1667))
	if err != nil {
		t.Fatal(err)
	}
	var back Params
	if err := json.Unmarshal(bz, &back); err != nil {
		t.Fatal(err)
	}
	if back.TargetAPYBps != 1667 || back.Mode() != EmissionModeTargetAPY {
		t.Fatalf("round trip lost the target: %s", bz)
	}

	info := DefaultEpochInfo()
	bz, err = json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if string(bz) != `{"current_epoch":0,"current_year":1,"block_start":0,"total_minted":"0"}` {
		t.Fatalf("epoch info without a recorded time changed its bytes: %s", bz)
	}
}

// The documented arithmetic: 1667 bps over a year at a reach of 0.47 mints
// bonded x 0.1667 / 0.47, and stakers receive 47% of it.
func TestTargetAPYEmissionArithmetic(t *testing.T) {
	bonded := math.NewInt(1_000_000_000_000_000) // 1B QOR in uqor
	reach := dec("0.47")
	year := TargetAPYEmission(bonded, 1667, reach, SecondsPerYear)
	want := dec("0.1667").MulInt(bonded).Quo(reach).TruncateInt()
	if !year.Equal(want) {
		t.Fatalf("a year mints %s, want %s", year, want)
	}
	toStakers := reach.MulInt(year).TruncateInt()
	gross := dec("0.1667").MulInt(bonded).TruncateInt()
	if diff := gross.Sub(toStakers).Abs(); diff.GT(math.NewInt(1)) {
		t.Fatalf("stakers receive %s, want %s", toStakers, gross)
	}
	// Net after a 10% commission is 15.0%.
	net := dec("0.9").MulInt(toStakers)
	if pct := net.QuoInt(bonded).MulInt64(10_000).TruncateInt64(); pct != 1500 {
		t.Fatalf("net yield %d bps, want 1500", pct)
	}

	// One mainnet epoch (a day) is 1/365 of the year, up to truncation.
	day := TargetAPYEmission(bonded, 1667, reach, 86_400)
	if d := day.MulRaw(365).Sub(year).Abs(); d.GT(math.NewInt(365)) {
		t.Fatalf("365 daily epochs mint %s, a year %s", day.MulRaw(365), year)
	}

	for name, got := range map[string]math.Int{
		"no bonded":   TargetAPYEmission(math.ZeroInt(), 1667, reach, 86_400),
		"no target":   TargetAPYEmission(bonded, 0, reach, 86_400),
		"no reach":    TargetAPYEmission(bonded, 1667, math.LegacyZeroDec(), 86_400),
		"no time":     TargetAPYEmission(bonded, 1667, reach, 0),
		"nil reach":   TargetAPYEmission(bonded, 1667, math.LegacyDec{}, 86_400),
		"nil bonded":  TargetAPYEmission(math.Int{}, 1667, reach, 86_400),
		"neg seconds": TargetAPYEmission(bonded, 1667, reach, -5),
	} {
		if !got.IsZero() {
			t.Fatalf("%s: got %s, want 0", name, got)
		}
	}
}

// ClampToCap is the clamp EpochEmissionScaled always applied, now shared with
// the target mode and the staking tier bonus.
func TestClampToCap(t *testing.T) {
	p := DefaultParams()
	if got := p.ClampToCap(math.NewInt(100), math.NewInt(5)); !got.Equal(math.NewInt(100)) {
		t.Fatalf("uncapped: got %s", got)
	}
	c := math.NewInt(1_000)
	p.MaxTotalEmission = &c
	if got := p.ClampToCap(math.NewInt(100), math.NewInt(950)); !got.Equal(math.NewInt(50)) {
		t.Fatalf("clamped: got %s, want 50", got)
	}
	if got := p.ClampToCap(math.NewInt(100), math.NewInt(1_000)); !got.IsZero() {
		t.Fatalf("at cap: got %s, want 0", got)
	}
	if rem, capped := p.RemainingCap(math.NewInt(1_200)); !capped || !rem.IsZero() {
		t.Fatalf("over cap: remaining %s capped %v", rem, capped)
	}
	if got := p.ClampToCap(math.NewInt(-3), math.ZeroInt()); !got.IsZero() {
		t.Fatalf("negative amount: got %s", got)
	}
	if got := p.NominalEpochSeconds(); got != 17_280*2 {
		t.Fatalf("nominal epoch seconds %d", got)
	}
}
