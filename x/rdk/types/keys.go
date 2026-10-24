package types

import "strconv"

const (
	ModuleName = "rdk"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	RollupConfigPrefix    = []byte("rdk/rollup/")
	SettlementBatchPrefix = []byte("rdk/batch/")
	LatestBatchPrefix     = []byte("rdk/lbatch/")
	DABlobPrefix          = []byte("rdk/blob/")
	LatestDAPrefix        = []byte("rdk/lda/")
	ChallengePrefix       = []byte("rdk/challenge/")
	ParamsKey             = []byte("rdk/params")
	WithdrawalDonePrefix  = []byte("rdk/wdone/")
	// WithdrawalLeafPrefix keys replay protection by the canonical withdrawal
	// leaf rather than by its position. The leaf binds rollup, index,
	// recipient, denom and amount, so the same withdrawal cannot be paid twice
	// by committing it again under another batch index.
	WithdrawalLeafPrefix = []byte("rdk/wleaf/")
	// RollupEscrowPrefix records how much of the pooled module balance belongs
	// to one rollup, so a batch on one rollup can never spend another's funds.
	RollupEscrowPrefix = []byte("rdk/escrow/")
	// FinalizeCursorPrefix records, per rollup, the index of the oldest batch
	// that is not final yet (from v3.2.5, FeatureRDKChallengeSettlement), so
	// auto-finalization does not read every batch a rollup ever settled.
	FinalizeCursorPrefix = []byte("rdk/fcursor/")
)

// FinalizeCursorKey returns the per-rollup finalization cursor key.
func FinalizeCursorKey(rollupID string) []byte {
	return append(append([]byte{}, FinalizeCursorPrefix...), []byte(rollupID)...)
}

// WithdrawalLeafDoneKey returns the replay-protection key for an executed
// withdrawal, identified by its canonical leaf digest.
func WithdrawalLeafDoneKey(rollupID string, leaf [32]byte) []byte {
	return append(append(append([]byte{}, WithdrawalLeafPrefix...), []byte(rollupID+"/")...), leaf[:]...)
}

// RollupEscrowKey returns the per-rollup escrow ledger key.
func RollupEscrowKey(rollupID string) []byte {
	return append(append([]byte{}, RollupEscrowPrefix...), []byte(rollupID)...)
}

// ChallengeKey returns the store key for a batch's challenge record.
func ChallengeKey(rollupID string, batchIndex uint64) []byte {
	return append(ChallengePrefix, []byte(rollupID+"/"+strconv.FormatUint(batchIndex, 10))...)
}

// WithdrawalDoneKey returns the replay-protection store key for an executed
// L2->L1 withdrawal, unique per (rollupID, batchIndex, withdrawalIndex).
func WithdrawalDoneKey(rollupID string, batchIndex, withdrawalIndex uint64) []byte {
	return append(WithdrawalDonePrefix,
		[]byte(rollupID+"/"+strconv.FormatUint(batchIndex, 10)+"/"+strconv.FormatUint(withdrawalIndex, 10))...)
}
