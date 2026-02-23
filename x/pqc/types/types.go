package types

// PQCAccountInfo stores the PQC keypair information for an account.
// v0.6.0: Migrated from DilithiumPubkey to generic PublicKey + AlgorithmID
// for multi-algorithm support.
type PQCAccountInfo struct {
	Address         string      `json:"address"`
	PublicKey       []byte      `json:"public_key"`                // PQC public key (algorithm-specific)
	AlgorithmID     AlgorithmID `json:"algorithm_id"`              // Which PQC algorithm this key uses
	ECDSAPubkey     []byte      `json:"ecdsa_pubkey,omitempty"`    // Classical ECDSA key for hybrid mode
	KeyType         string      `json:"key_type"`                  // "hybrid" | "pqc_only" | "classical_only"
	CreatedAtHeight int64       `json:"created_at_height"`

	// Migration fields — set when the account is in dual-key mode during migration
	MigrationPublicKey   []byte      `json:"migration_public_key,omitempty"`    // New algorithm public key
	MigrationAlgorithmID AlgorithmID `json:"migration_algorithm_id,omitempty"`  // New algorithm ID
}

// PQCStats tracks module-level statistics.
type PQCStats struct {
	TotalPQCVerifications   uint64 `json:"total_pqc_verifications"`
	TotalClassicalFallbacks uint64 `json:"total_classical_fallbacks"`
	TotalMLKEMOperations    uint64 `json:"total_mlkem_operations"`
	TotalDualSigVerifies    uint64 `json:"total_dual_sig_verifies"`
	TotalKeyMigrations      uint64 `json:"total_key_migrations"`
}

// Key type constants.
const (
	KeyTypeHybrid        = "hybrid"
	KeyTypePQCOnly       = "pqc_only"
	KeyTypeClassicalOnly = "classical_only"
)
