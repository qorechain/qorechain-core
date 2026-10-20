package types

import (
	"testing"

	"cosmossdk.io/math"
)

// QSR-2026-0377: the voting-power multiplier is capped at 2.0x and every lock
// tier sits between 1 and the governance multiplier, non-decreasing.
func TestParamsMultiplierCap(t *testing.T) {
	if err := DefaultParams().Validate(); err != nil {
		t.Fatalf("default params must validate: %v", err)
	}

	p := DefaultParams()
	p.GovernanceMultiplier = math.LegacyNewDec(100)
	if err := p.Validate(); err == nil {
		t.Fatal("a 100x governance multiplier must be refused")
	}

	p = DefaultParams()
	p.GovernanceMultiplier = math.LegacyMustNewDecFromStr("0.5")
	if err := p.Validate(); err == nil {
		t.Fatal("a governance multiplier below 1 must be refused")
	}

	p = DefaultParams()
	p.LockMultiplierSchedule[4].Multiplier = math.LegacyMustNewDecFromStr("2.5")
	if err := p.Validate(); err == nil {
		t.Fatal("a lock tier above the governance multiplier must be refused")
	}

	p = DefaultParams()
	p.LockMultiplierSchedule[2].Multiplier = math.LegacyMustNewDecFromStr("1.1")
	if err := p.Validate(); err == nil {
		t.Fatal("a lock tier lower than the previous one must be refused")
	}

	p = DefaultParams()
	p.LockMultiplierSchedule[0].Multiplier = math.LegacyMustNewDecFromStr("0.9")
	if err := p.Validate(); err == nil {
		t.Fatal("a lock tier below 1 must be refused")
	}

	// The mainnet and diana genesis values: governance 2.0, tiers 1.0..2.0.
	p = DefaultParams()
	p.GovernanceMultiplier = math.LegacyMustNewDecFromStr("2.000000000000000000")
	if err := p.Validate(); err != nil {
		t.Fatalf("shipped genesis values must validate: %v", err)
	}
}
