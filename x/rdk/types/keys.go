package types

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
	ParamsKey             = []byte("rdk/params")
)
