package types

const (
	ModuleName = "fairblock"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey         = []byte("fairblock/config")
	EncryptedTxPrefix = []byte("fairblock/etx/")
	DecryptionPrefix  = []byte("fairblock/dec/")
)
