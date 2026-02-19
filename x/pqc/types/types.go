package types

// PQCAccountInfo stores the PQC keypair information for an account.
type PQCAccountInfo struct {
	Address          string `json:"address"`
	DilithiumPubkey  []byte `json:"dilithium_pubkey"`
	ECDSAPubkey      []byte `json:"ecdsa_pubkey,omitempty"`
	KeyType          string `json:"key_type"` // "hybrid" | "pqc_only" | "classical_only"
	CreatedAtHeight  int64  `json:"created_at_height"`
}

// PQCStats tracks module-level statistics.
type PQCStats struct {
	TotalPQCVerifications     uint64 `json:"total_pqc_verifications"`
	TotalClassicalFallbacks   uint64 `json:"total_classical_fallbacks"`
	TotalMLKEMOperations      uint64 `json:"total_mlkem_operations"`
}

// Key type constants.
const (
	KeyTypeHybrid         = "hybrid"
	KeyTypePQCOnly        = "pqc_only"
	KeyTypeClassicalOnly  = "classical_only"
)
