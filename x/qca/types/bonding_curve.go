package types

// BondingCurveConfig holds bonding curve parameters.
// Formula: R(v,t) = Beta * S_v * (1 + Alpha * log(1 + L_v)) * Q(r_v) * P(t)
type BondingCurveConfig struct {
	Alpha           string `json:"alpha"`            // loyalty sensitivity (LegacyDec, default: "0.1")
	Beta            string `json:"beta"`             // base multiplier (LegacyDec, default: "1.0")
	PhaseMultiplier string `json:"phase_multiplier"` // protocol phase multiplier (LegacyDec, default: "1.5" for genesis)
}

// DefaultBondingCurveConfig returns the default bonding curve configuration.
func DefaultBondingCurveConfig() BondingCurveConfig {
	return BondingCurveConfig{
		Alpha:           "0.1",
		Beta:            "1.0",
		PhaseMultiplier: "1.5",
	}
}
