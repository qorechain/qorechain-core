package types

import (
	"math"
	"testing"
)

// A governance config update must not store a threshold that compares false
// with every bound (NaN) or lies outside [0, 1].
func TestAIConfigValidateRejectsNaNAndOutOfRange(t *testing.T) {
	ok := DefaultAIConfig()
	if err := ok.Validate(); err != nil {
		t.Fatalf("default config refused: %v", err)
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -0.1, 1.1} {
		c := DefaultAIConfig()
		c.AnomalyThreshold = v
		if c.Validate() == nil {
			t.Fatalf("anomaly_threshold %v accepted", v)
		}
		c = DefaultAIConfig()
		c.RiskThreshold = v
		if c.Validate() == nil {
			t.Fatalf("risk_threshold %v accepted", v)
		}
	}
	m := MsgUpdateConfig{AnomalyThreshold: "NaN", RiskThreshold: "0.8", RoutingStrategy: "round_robin"}
	if _, err := m.ToConfig(); err == nil {
		t.Fatal("MsgUpdateConfig with a NaN threshold accepted")
	}
}
