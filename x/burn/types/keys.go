package types

const (
	ModuleName = "burn"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ParamsKey        = []byte("burn/params")
	TotalBurnedKey   = []byte("burn/total-burned")
	BurnRecordPrefix = []byte("burn/records/")
	BurnStatsPrefix  = []byte("burn/stats/")
)
