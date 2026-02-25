package types

const (
	ModuleName = "inflation"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ParamsKey         = []byte("inflation/params")
	CurrentEpochKey   = []byte("inflation/epoch")
	EmissionLogPrefix = []byte("inflation/emissions/") // reserved for future emission log indexing
)

const MaxExportEmissions = 1_000_000
