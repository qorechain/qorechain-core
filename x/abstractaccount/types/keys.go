package types

const (
	ModuleName = "abstractaccount"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey         = []byte("aa/config")
	AccountPrefix     = []byte("aa/acc/")
	SessionKeyPrefix  = []byte("aa/session/")
)
