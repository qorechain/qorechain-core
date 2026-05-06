package types

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
)

// TestDefaultParams_Validate ensures DefaultParams pass validation.
func TestDefaultParams_Validate(t *testing.T) {
	p := DefaultParams()
	if err := p.Validate(); err != nil {
		t.Fatalf("DefaultParams should validate but got: %v", err)
	}
}

// TestDefaultParams_FeeSharesSumToOne is the regression test for the v2.6.3
// QCTokenomics v2 alignment fix. The old defaults summed to >1.0 (39% + 29.5%
// + 19.5% + 9% + 3% = 100% but the burn was indirectly counted). This locks
// in: ValidatorShare + GasBurnRate + TreasuryShare + StakerShare + LightNodeShare = 1.0.
func TestDefaultParams_FeeSharesSumToOne(t *testing.T) {
	p := DefaultParams()
	sum := p.ValidatorShare.
		Add(p.GasBurnRate).
		Add(p.TreasuryShare).
		Add(p.StakerShare).
		Add(p.LightNodeShare)
	if !sum.Equal(math.LegacyOneDec()) {
		t.Fatalf("default fee shares sum != 1.0, got %s (validator=%s burn=%s treasury=%s stakers=%s lightnode=%s)",
			sum, p.ValidatorShare, p.GasBurnRate, p.TreasuryShare, p.StakerShare, p.LightNodeShare)
	}
}

// TestDefaultParams_QCTokenomicsV2Alignment locks in the exact v2 split
// so a future change to any individual share is caught here.
func TestDefaultParams_QCTokenomicsV2Alignment(t *testing.T) {
	p := DefaultParams()
	cases := []struct {
		name     string
		actual   math.LegacyDec
		expected math.LegacyDec
	}{
		{"validator", p.ValidatorShare, math.LegacyNewDecWithPrec(37, 2)}, // 37%
		{"burn", p.GasBurnRate, math.LegacyNewDecWithPrec(30, 2)},         // 30%
		{"treasury", p.TreasuryShare, math.LegacyNewDecWithPrec(20, 2)},   // 20%
		{"staker", p.StakerShare, math.LegacyNewDecWithPrec(10, 2)},       // 10%
		{"lightnode", p.LightNodeShare, math.LegacyNewDecWithPrec(3, 2)},  // 3%
	}
	for _, c := range cases {
		if !c.actual.Equal(c.expected) {
			t.Errorf("%s share: got %s, want %s", c.name, c.actual, c.expected)
		}
	}
}

// TestParams_Validate_RejectsBrokenSum verifies Validate catches share-sum drift.
func TestParams_Validate_RejectsBrokenSum(t *testing.T) {
	p := DefaultParams()
	// Bump validator share by 1% — total becomes 1.01.
	p.ValidatorShare = p.ValidatorShare.Add(math.LegacyNewDecWithPrec(1, 2))
	err := p.Validate()
	if err == nil {
		t.Fatal("expected validation error when shares don't sum to 1.0")
	}
	if !strings.Contains(err.Error(), "must sum to 1.0") {
		t.Errorf("expected sum-to-1 error, got: %v", err)
	}
}

// TestParams_Validate_RejectsNegativeShare verifies negative LightNodeShare is caught.
func TestParams_Validate_RejectsNegativeShare(t *testing.T) {
	p := DefaultParams()
	p.LightNodeShare = math.LegacyNewDecWithPrec(-3, 2)
	if err := p.Validate(); err == nil {
		t.Fatal("expected validation error for negative light_node_share")
	}
}

// TestParams_Validate_RejectsShareOverOne verifies caps at 1.0.
func TestParams_Validate_RejectsShareOverOne(t *testing.T) {
	p := DefaultParams()
	p.LightNodeShare = math.LegacyNewDecWithPrec(150, 2) // 1.5
	if err := p.Validate(); err == nil {
		t.Fatal("expected validation error for light_node_share > 1.0")
	}
}

// TestMilestoneBurnSchedule_Default verifies the default tier ordering.
func TestMilestoneBurnSchedule_Default(t *testing.T) {
	p := DefaultParams()
	if len(p.MilestoneBurnSchedule) < 2 {
		t.Fatalf("expected at least 2 milestone tiers, got %d", len(p.MilestoneBurnSchedule))
	}
	for i := 1; i < len(p.MilestoneBurnSchedule); i++ {
		if p.MilestoneBurnSchedule[i].TxThreshold <= p.MilestoneBurnSchedule[i-1].TxThreshold {
			t.Errorf("milestone tiers not strictly increasing at index %d: %d <= %d",
				i, p.MilestoneBurnSchedule[i].TxThreshold, p.MilestoneBurnSchedule[i-1].TxThreshold)
		}
	}
}

// TestParams_Validate_RejectsNonIncreasingMilestones verifies schedule ordering enforcement.
func TestParams_Validate_RejectsNonIncreasingMilestones(t *testing.T) {
	p := DefaultParams()
	// Swap so second tier <= first tier.
	p.MilestoneBurnSchedule = []MilestoneBurnTier{
		{TxThreshold: 1_000_000, BurnAmount: math.NewInt(1_000_000_000_000)},
		{TxThreshold: 1_000_000, BurnAmount: math.NewInt(2_000_000_000_000)},
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected validation error for non-strictly-increasing milestone thresholds")
	}
}

// TestParams_Validate_RejectsZeroMilestoneAmount verifies positive amount enforcement.
func TestParams_Validate_RejectsZeroMilestoneAmount(t *testing.T) {
	p := DefaultParams()
	p.MilestoneBurnSchedule = []MilestoneBurnTier{
		{TxThreshold: 1_000_000, BurnAmount: math.NewInt(0)},
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected validation error for zero milestone burn amount")
	}
}
