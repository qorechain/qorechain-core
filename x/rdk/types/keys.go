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
)

// ChallengeKey returns the store key for a batch's challenge record.
func ChallengeKey(rollupID string, batchIndex uint64) []byte {
	return append(ChallengePrefix, []byte(rollupID+"/"+strconv.FormatUint(batchIndex, 10))...)
}
