package types

// SVMTxRecord is a stored record of an executed SVM program transaction. It
// backs the Solana-compatible getSignaturesForAddress and getTransaction RPC
// methods. Records are kept in a bounded, recent window (see the keeper's
// tx-history window) rather than as full archival history — long-range history
// is the responsibility of the external indexer.
type SVMTxRecord struct {
	// Signature is the host chain transaction hash (uppercase hex), used as the
	// Solana-style transaction signature.
	Signature string `json:"signature"`
	// Slot is the SVM slot (chain height) at which the transaction executed.
	Slot uint64 `json:"slot"`
	// Seq is the monotonic insertion sequence (used for windowed pruning).
	Seq uint64 `json:"seq"`
	// BlockTime is the unix timestamp of the executing block.
	BlockTime int64 `json:"block_time"`
	// Sender is the base58 SVM address of the fee payer / primary signer.
	Sender string `json:"sender"`
	// ProgramID is the base58 address of the invoked program.
	ProgramID string `json:"program_id"`
	// Accounts are the base58 addresses of the instruction's input accounts.
	Accounts []string `json:"accounts"`
	// Success reports whether the execution succeeded.
	Success bool `json:"success"`
	// ComputeUnits is the compute budget consumed by the execution.
	ComputeUnits uint64 `json:"compute_units"`
	// Logs are the program log messages emitted during execution.
	Logs []string `json:"logs,omitempty"`
	// Err holds the error string when Success is false.
	Err string `json:"err,omitempty"`
}
