package types

const (
	ModuleName = "qca"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey = []byte("qca/config")
	StatsKey  = []byte("qca/stats")
)
