package types

const (
	ModuleName = "xqore"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ParamsKey           = []byte("xqore/params")
	PositionPrefix      = []byte("xqore/positions/")
	TotalLockedKey      = []byte("xqore/total-locked")
	TotalXQOREKey       = []byte("xqore/total-supply")
	RebaseHistoryPrefix = []byte("xqore/rebases/") // reserved for future rebase indexing
)

const MaxExportPositions = 1_000_000
