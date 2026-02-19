package types

// Params defines the parameters for the x/pqc module.
type Params struct {
	PQCPrimary             bool  `json:"pqc_primary"`
	AllowClassicalFallback bool  `json:"allow_classical_fallback"`
	MinSecurityLevel       int32 `json:"min_security_level"`
}

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return Params{
		PQCPrimary:             true,
		AllowClassicalFallback: true,  // Phase 3: allow classical fallback
		MinSecurityLevel:       5,     // NIST Level 5 (Dilithium-5)
	}
}
