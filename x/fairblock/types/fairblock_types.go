package types

import "time"

// FairBlockConfig holds tIBE configuration.
type FairBlockConfig struct {
	Enabled          bool  `json:"enabled"`
	TIBEThreshold    int   `json:"tibe_threshold"`
	DecryptionDelay  int64 `json:"decryption_delay"`
	MaxEncryptedSize int64 `json:"max_encrypted_size"`
}

// DefaultFairBlockConfig returns default config (disabled).
func DefaultFairBlockConfig() FairBlockConfig {
	return FairBlockConfig{
		Enabled:          false,
		TIBEThreshold:    5,
		DecryptionDelay:  3,
		MaxEncryptedSize: 65536,
	}
}

// EncryptedTx represents an encrypted transaction in the mempool.
type EncryptedTx struct {
	ID            string    `json:"id"`
	EncryptedData []byte    `json:"encrypted_data"`
	Sender        string    `json:"sender"`
	TargetHeight  int64     `json:"target_height"`
	SubmittedAt   time.Time `json:"submitted_at"`
	DecryptedData []byte    `json:"decrypted_data,omitempty"`
	Decrypted     bool      `json:"decrypted"`
}

// DecryptionShare represents a validator decryption share.
type DecryptionShare struct {
	Validator string `json:"validator"`
	TxID      string `json:"tx_id"`
	ShareData []byte `json:"share_data"`
	Height    int64  `json:"height"`
}
