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

	// SpendAccumPrefix keys the per-authenticator daily-spend accumulator used to
	// enforce SpendingRule.DailyLimit (v3.1.84). Value = big-endian uint64 of the
	// cumulative base-unit amount spent that UTC day. The day comes from
	// ctx.BlockTime() so it is deterministic across validators.
	SpendAccumPrefix = []byte("aa/spend/")
)

// AuthIndexKey builds the reverse-index key for a (scheme, pubkey) authenticator.
func AuthIndexKey(scheme string, pubkey []byte) []byte {
	key := append([]byte{}, AuthIndexPrefix...)
	key = append(key, []byte(scheme)...)
	key = append(key, '/')
	return append(key, pubkey...)
}

// SpendAccumKey builds the daily-spend accumulator key for
// (account, authenticator pubkey, denom, UTC day). dayUnix = blockTimeUnix / 86400.
func SpendAccumKey(account string, pubkey []byte, denom string, dayUnix int64) []byte {
	key := append([]byte{}, SpendAccumPrefix...)
	key = append(key, []byte(account)...)
	key = append(key, '/')
	key = append(key, pubkey...)
	key = append(key, '/')
	key = append(key, []byte(denom)...)
	key = append(key, '/')
	var d [8]byte
	for i := 7; i >= 0; i-- {
		d[i] = byte(dayUnix)
		dayUnix >>= 8
	}
	return append(key, d[:]...)
}
