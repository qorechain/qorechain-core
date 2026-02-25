package types

const (
	ModuleName = "burn"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// MaxExportRecords is the upper bound on burn records exported in genesis.
const MaxExportRecords = 1_000_000

var (
	ParamsKey        = []byte("burn/params")
	TotalBurnedKey   = []byte("burn/total-burned")
	BurnRecordPrefix = []byte("burn/records/")
	BurnStatsPrefix  = []byte("burn/stats/") // reserved for future per-source stats indexing
)
