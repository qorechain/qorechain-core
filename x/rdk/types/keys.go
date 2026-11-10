package types

import (
	"encoding/binary"
	"strconv"
)

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
	// BatchCountPrefix records, per rollup, the height of the last block it
	// settled a batch in and how many it settled there (from v3.2.8,
	// FeatureRDKMaxBatchesPerBlock).
	BatchCountPrefix = []byte("rdk/bcount/")
	// CompactCursorPrefix records, per rollup, the index of the next batch
	// finalized-batch compaction looks at (from v3.2.8,
	// FeatureRDKBatchCompaction).
	CompactCursorPrefix = []byte("rdk/ccursor/")

	// The v3.2.8 posted-data and inbox keys (FeatureRDKPostRollupData). The
	// rollup id inside them is length-prefixed (PostedDataRollupKey), so no
	// id can be read as part of another one.
	//
	// PostedDataPrefix holds one commitment per post: <id>/<seq be64>.
	PostedDataPrefix = []byte("rdk/pdata/")
	// PostedDataHashPrefix maps <id>/<sha256> to the sequence of the latest
	// retained post with that hash.
	PostedDataHashPrefix = []byte("rdk/pdhash/")
	// PostedDataExpiryPrefix is the pruning queue: <expiry be64>/<id>/<seq be64>.
	PostedDataExpiryPrefix = []byte("rdk/pdexp/")
	// InboxPrefix holds each rollup's inbox state.
	InboxPrefix = []byte("rdk/inbox/")
	// PostedDataBlockKey records the height of the last block with a post and
	// the billable bytes posted in it.
	PostedDataBlockKey = []byte("rdk/pdblock")
)

// PostedDataRollupKey is the length-prefixed rollup id used inside the
// posted-data keys.
func PostedDataRollupKey(rollupID string) []byte {
	out := make([]byte, 2, 2+len(rollupID))
	binary.BigEndian.PutUint16(out, uint16(len(rollupID)))
	return append(out, rollupID...)
}

func be64(n uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], n)
	return b[:]
}

// PostedDataKey returns the commitment key of one post.
func PostedDataKey(rollupID string, seq uint64) []byte {
	k := append(append([]byte{}, PostedDataPrefix...), PostedDataRollupKey(rollupID)...)
	return append(k, be64(seq)...)
}

// PostedDataRollupPrefix returns the prefix of every commitment of a rollup.
func PostedDataRollupPrefix(rollupID string) []byte {
	return append(append([]byte{}, PostedDataPrefix...), PostedDataRollupKey(rollupID)...)
}

// PostedDataHashKey returns the hash-index key of a post.
func PostedDataHashKey(rollupID string, hash []byte) []byte {
	k := append(append([]byte{}, PostedDataHashPrefix...), PostedDataRollupKey(rollupID)...)
	return append(k, hash...)
}

// PostedDataExpiryKey returns the pruning-queue key of a post.
func PostedDataExpiryKey(expiry int64, rollupID string, seq uint64) []byte {
	k := append(append([]byte{}, PostedDataExpiryPrefix...), be64(uint64(expiry))...)
	k = append(k, PostedDataRollupKey(rollupID)...)
	return append(k, be64(seq)...)
}

// ParsePostedDataExpiryKey splits a pruning-queue key (without its prefix)
// into expiry height, rollup id and sequence.
func ParsePostedDataExpiryKey(rest []byte) (int64, string, uint64, bool) {
	if len(rest) < 8+2+8 {
		return 0, "", 0, false
	}
	expiry := int64(binary.BigEndian.Uint64(rest[:8]))
	n := int(binary.BigEndian.Uint16(rest[8:10]))
	if len(rest) != 10+n+8 {
		return 0, "", 0, false
	}
	id := string(rest[10 : 10+n])
	seq := binary.BigEndian.Uint64(rest[10+n:])
	return expiry, id, seq, true
}

// InboxKey returns a rollup's inbox state key.
func InboxKey(rollupID string) []byte {
	return append(append([]byte{}, InboxPrefix...), PostedDataRollupKey(rollupID)...)
}

// BatchCountKey returns the per-rollup batches-in-this-block counter key.
func BatchCountKey(rollupID string) []byte {
	return append(append([]byte{}, BatchCountPrefix...), []byte(rollupID)...)
}

// CompactCursorKey returns the per-rollup compaction cursor key.
func CompactCursorKey(rollupID string) []byte {
	return append(append([]byte{}, CompactCursorPrefix...), []byte(rollupID)...)
}

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
