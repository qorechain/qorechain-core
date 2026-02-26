package types

// BatchStatus represents the lifecycle state of a settlement batch.
type BatchStatus string

const (
	BatchSubmitted  BatchStatus = "submitted"
	BatchChallenged BatchStatus = "challenged"
	BatchFinalized  BatchStatus = "finalized"
	BatchRejected   BatchStatus = "rejected"
)

// SettlementBatch represents a rollup state batch submitted for settlement.
type SettlementBatch struct {
	RollupID      string        `json:"rollup_id"`
	BatchIndex    uint64        `json:"batch_index"`
	StateRoot     []byte        `json:"state_root"`
	PrevStateRoot []byte        `json:"prev_state_root"`
	TxCount       uint64        `json:"tx_count"`
	DataHash      []byte        `json:"data_hash"`
	ProofType     ProofSystem   `json:"proof_type"`
	Proof         []byte        `json:"proof,omitempty"`
	SequencerMode SequencerMode `json:"sequencer_mode"`
	L1BlockRange  [2]int64      `json:"l1_block_range"` // For based rollups: L1 block range
	SubmittedAt   int64         `json:"submitted_at"`   // Block height
	FinalizedAt   int64         `json:"finalized_at"`   // Block height
	Status        BatchStatus   `json:"status"`
}
