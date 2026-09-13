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

	// AuthNoncePrefix keys the per-authenticator monotonic sequence used for
	// Native-lane (MsgExecuteCosmos) replay protection (v3.1.85). Value =
	// big-endian uint64 of the next expected sequence for (account, authenticator
	// pubkey). Unlike the EVM lane — which reuses the account's auto-incrementing
	// EVM nonce — a bank send increments nothing, so the module tracks the
	// sequence itself: the signed nonce must equal the stored value, which is
	// bumped by one on each successful execution.
	AuthNoncePrefix = []byte("aa/authnonce/")
)

// AuthIndexKey builds the reverse-index key for a (scheme, pubkey) authenticator.
func AuthIndexKey(scheme string, pubkey []byte) []byte {
	key := append([]byte{}, AuthIndexPrefix...)
	key = append(key, []byte(scheme)...)
	key = append(key, '/')
	return append(key, pubkey...)
}

// SpendAccumKey builds the daily-spend accumulator key for
// (account, scope, denom, UTC day). dayUnix = blockTimeUnix / 86400.
//
// SCOPE MATTERS AND IS NOT ALWAYS THE KEY. The accumulator used to be keyed on the
// authenticator's public key unconditionally, so an account with three linked keys
// had three separate daily allowances and could spend three times its stated daily
// limit. That is correct when the limit was set ON a key, and wrong when it was set
// on the ACCOUNT: an owner who says "this account may spend 1,000 a day" has said
// something about the account, not about each key they happen to add later.
//
// The scope therefore follows the rule: a rule defined on the account accumulates
// against the account (pass a nil scope), a rule defined on one key accumulates
// against that key. See ResolveSpendRule, which is what decides which rule applies.
func SpendAccumKey(account string, scope []byte, denom string, dayUnix int64) []byte {
	key := append([]byte{}, SpendAccumPrefix...)
	key = append(key, []byte(account)...)
	key = append(key, '/')
	// A nil scope is the account-wide bucket. It cannot collide with a key-scoped
	// one, because a public key is never empty.
	key = append(key, scope...)
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

// AuthNonceKey builds the per-authenticator sequence key for (account, pubkey).
func AuthNonceKey(account string, pubkey []byte) []byte {
	key := append([]byte{}, AuthNoncePrefix...)
	key = append(key, []byte(account)...)
	key = append(key, '/')
	return append(key, pubkey...)
}
