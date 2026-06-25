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

	// Hybrid signature params (v1.1.0)
	HybridSignatureMode HybridSignatureMode `json:"hybrid_signature_mode"` // Chain-wide hybrid sig enforcement
}

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return Params{
		PQCPrimary:             true,
		AllowClassicalFallback: false, // PQC-first: no classical-only cosmos txs
		MinSecurityLevel:       5,     // NIST Level 5 (Dilithium-5)
		DefaultMigrationBlocks: DefaultMigrationBlocks,
		DefaultSignatureAlgo:   AlgorithmDilithium5,
		// PQC-required by default: every cosmos account must carry a Dilithium-5
		// hybrid signature (in addition to the unavoidable secp256k1 sig). PQC key
		// registration/migration txs are exempt (bootstrap). EVM txs use a separate
		// ante path. Hybrid is the on-chain norm; the classical sig is what external
		// networks verify at bridge egress.
		HybridSignatureMode: HybridRequired,
	}
}
