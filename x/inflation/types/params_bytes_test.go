package types

import (
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/math"
)

// mainnetGenesisParams is the x/inflation params block exactly as it appears in
// the live mainnet genesis file: no emission_mode, no fixed_epoch_emission, no
// max_total_emission, because none of them existed when it was written.
const mainnetGenesisParams = `{"schedule":[{"year":1,"inflation_rate":"0.175000000000000000"},` +
	`{"year":2,"inflation_rate":"0.110000000000000000"},` +
	`{"year":3,"inflation_rate":"0.070000000000000000"},` +
	`{"year":4,"inflation_rate":"0.070000000000000000"},` +
	`{"year":5,"inflation_rate":"0.020000000000000000"}],` +
	`"epoch_length":43200,"enabled":true}`

// The consensus property. SetParams is called by InitGenesis, so if adding
// fields changes the bytes written for a record that does not use them, a node
// syncing from genesis computes a different hash at height 0 and cannot join.
// This is exactly the failure that took node onboarding down for days; do not
// let a value type back into this struct.
func TestParamsFromMainnetGenesisRoundTripByteIdentical(t *testing.T) {
	var p Params
	if err := json.Unmarshal([]byte(mainnetGenesisParams), &p); err != nil {
		t.Fatalf("real mainnet genesis params must unmarshal: %v", err)
	}

	out, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != mainnetGenesisParams {
		t.Fatalf("re-serialising the mainnet genesis params changed the bytes.\n"+
			" was: %s\n now: %s", mainnetGenesisParams, out)
	}
}

// The same record must also survive validation. Normalising on read is not
// enough: Validate rejected the real mainnet genesis once already.
func TestMainnetGenesisParamsValidate(t *testing.T) {
	var p Params
	if err := json.Unmarshal([]byte(mainnetGenesisParams), &p); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("live mainnet params must validate, got %v", err)
	}
	if p.Mode() != EmissionModeSchedule {
		t.Fatalf("absent emission_mode must mean %q, got %q", EmissionModeSchedule, p.Mode())
	}
	if !p.FixedEmission().IsZero() || !p.EmissionCap().IsZero() {
		t.Fatal("absent optional fields must read as zero, not panic or garbage")
	}
}

// And the fields must still work when they are actually used.
func TestFixedModeRoundTrips(t *testing.T) {
	emission := math.NewInt(16_239_000_000)
	cap_ := math.NewInt(114_285_714_000_000)
	p := Params{
		EpochLength:        43200,
		Enabled:            true,
		EmissionMode:       EmissionModeFixed,
		FixedEpochEmission: &emission,
		MaxTotalEmission:   &cap_,
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("fixed-mode params must validate: %v", err)
	}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var back Params
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !back.FixedEmission().Equal(emission) || !back.EmissionCap().Equal(cap_) {
		t.Fatalf("values did not survive the round trip: %s", b)
	}
}

// A zero value must not be stored as a set field — that would write the key and
// change the bytes for no gain, which is the bug this whole change removes.
func TestZeroOptionalIsNotStored(t *testing.T) {
	p := Params{
		Schedule:           DefaultParams().Schedule,
		EpochLength:        43200,
		Enabled:            true,
		FixedEpochEmission: OptionalInt(math.ZeroInt()),
		MaxTotalEmission:   OptionalInt(math.Int{}),
	}
	b, _ := json.Marshal(p)
	for _, key := range []string{"fixed_epoch_emission", "max_total_emission", "emission_mode"} {
		if strings.Contains(string(b), key) {
			t.Fatalf("%s must be absent when unset, got %s", key, b)
		}
	}
}
