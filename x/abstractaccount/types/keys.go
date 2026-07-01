package types

const (
	ModuleName = "abstractaccount"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey        = []byte("aa/config")
	AccountPrefix    = []byte("aa/acc/")
	SessionKeyPrefix = []byte("aa/session/")
	// AuthIndexPrefix is the reverse index scheme|pubkey -> canonical account
	// address, used to resolve a signature from any wallet (e.g. Phantom) back
	// to the single account it authenticates.
	AuthIndexPrefix = []byte("aa/authidx/")
)

// AuthIndexKey builds the reverse-index key for a (scheme, pubkey) authenticator.
func AuthIndexKey(scheme string, pubkey []byte) []byte {
	key := append([]byte{}, AuthIndexPrefix...)
	key = append(key, []byte(scheme)...)
	key = append(key, '/')
	return append(key, pubkey...)
}
