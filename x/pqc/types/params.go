package types

// DefaultMigrationBlocks is the default dual-signature migration period.
// 1,000,000 blocks ≈ ~69 days at 6s/block.
const DefaultMigrationBlocks int64 = 1_000_000

// Params defines the parameters for the x/pqc module.
type Params struct {
	PQCPrimary             bool  `json:"pqc_primary"`
	AllowClassicalFallback bool  `json:"allow_classical_fallback"`
	MinSecurityLevel       int32 `json:"min_security_level"`

	// Algorithm agility params (v0.6.0)
	DefaultMigrationBlocks int64       `json:"default_migration_blocks"` // Dual-sig period in blocks
	DefaultSignatureAlgo   AlgorithmID `json:"default_signature_algo"`   // Default sig algorithm for new keys
}

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return Params{
		PQCPrimary:             true,
		AllowClassicalFallback: true,              // Allow classical ECDSA fallback
		MinSecurityLevel:       5,                 // NIST Level 5 (Dilithium-5)
		DefaultMigrationBlocks: DefaultMigrationBlocks,
		DefaultSignatureAlgo:   AlgorithmDilithium5,
	}
}
