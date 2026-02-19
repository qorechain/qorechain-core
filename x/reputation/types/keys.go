package types

const (
	ModuleName = "reputation"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ValidatorPrefix = []byte("reputation/validators/")
	ParamsKey       = []byte("reputation/params")
	HistoryPrefix   = []byte("reputation/history/")
)
