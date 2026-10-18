package types

// PQCAccountInfo stores the PQC keypair information for an account.
// v0.6.0: Migrated from DilithiumPubkey to generic PublicKey + AlgorithmID
// for multi-algorithm support.
type PQCAccountInfo struct {
	Address         string      `json:"address"`
	PublicKey       []byte      `json:"public_key"`             // PQC public key (algorithm-specific)
	AlgorithmID     AlgorithmID `json:"algorithm_id"`           // Which PQC algorithm this key uses
	ECDSAPubkey     []byte      `json:"ecdsa_pubkey,omitempty"` // Classical ECDSA key for hybrid mode
	KeyType         string      `json:"key_type"`               // "hybrid" | "pqc_only" | "classical_only"
	CreatedAtHeight int64       `json:"created_at_height"`

	// Migration fields — set when the account is in dual-key mode during migration
	MigrationPublicKey   []byte      `json:"migration_public_key,omitempty"`   // New algorithm public key
	MigrationAlgorithmID AlgorithmID `json:"migration_algorithm_id,omitempty"` // New algorithm ID

	// RetiredKeyHashes are SHA-256 hashes of the keys this account rotated or
	// migrated away from (FeaturePQCRotationReplayGuard). It may not return to
	// one, so a rotation signed before a key cycle cannot be replayed. Empty
	// and omitted for every record written before the switch.
	RetiredKeyHashes [][]byte `json:"retired_key_hashes,omitempty"`

	// PossessionProven says the current key was proven by a proof of
	// possession (FeaturePQCPossessionFlag): a registration that carried a
	// valid one, or a rotation or migration, which verify a signature by the
	// new key. A registration without one leaves it false. Omitted while
	// false, so every record written before the switch keeps its bytes.
	//
	// A key registered without a proof may be a key whose secret nobody
	// holds, copied from somewhere else. Anything that treats a registered key
	// as the account's own (the pqcKeyStatus and pqcVerify precompiles, if
	// they are ever activated for contracts) must check this flag first.
	PossessionProven bool `json:"possession_proven,omitempty"`
}

// PQCStats tracks module-level statistics.
type PQCStats struct {
	TotalPQCVerifications    uint64 `json:"total_pqc_verifications"`
	TotalClassicalFallbacks  uint64 `json:"total_classical_fallbacks"`
	TotalMLKEMOperations     uint64 `json:"total_mlkem_operations"`
	TotalDualSigVerifies     uint64 `json:"total_dual_sig_verifies"`
	TotalKeyMigrations       uint64 `json:"total_key_migrations"`
	TotalHybridVerifications uint64 `json:"total_hybrid_verifications"` // v1.1.0
}

// Key type constants.
const (
	KeyTypeHybrid        = "hybrid"
	KeyTypePQCOnly       = "pqc_only"
	KeyTypeClassicalOnly = "classical_only"
)
