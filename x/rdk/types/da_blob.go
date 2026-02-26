package types

import "time"

// DABlob represents a data availability blob stored on-chain.
type DABlob struct {
	RollupID  string    `json:"rollup_id"`
	BlobIndex uint64    `json:"blob_index"`
	Data      []byte    `json:"data"`
	Commitment []byte   `json:"commitment"`
	Height    int64     `json:"height"`
	Namespace []byte    `json:"namespace,omitempty"`
	StoredAt  time.Time `json:"stored_at"`
	Pruned    bool      `json:"pruned"`
}

// DACommitment represents a commitment to a DA blob.
type DACommitment struct {
	RollupID  string    `json:"rollup_id"`
	BlobIndex uint64    `json:"blob_index"`
	Backend   DABackend `json:"backend"`
	Hash      []byte    `json:"hash"`
	Size      uint64    `json:"size"`
	Confirmed bool      `json:"confirmed"`
}
