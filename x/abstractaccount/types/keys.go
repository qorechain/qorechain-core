package types

import "encoding/binary"

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

	// SpendAccumIntPrefix keys the daily-spend accumulator of a denom that has
	// its own SpendingRule.DenomLimits entry (v3.2.2). The key layout after the
	// prefix is SpendAccumKey's; the value is the decimal string of the amount,
	// because token amounts do not fit the legacy uint64 bucket. It is a separate
	// prefix so the legacy buckets keep their bytes and their export format.
	SpendAccumIntPrefix = []byte("aa/spendx/")
)

// SpendAccumIntKey builds the arbitrary-precision daily-spend key for
// (account, scope, denom, UTC day), with SpendAccumKey's layout.
func SpendAccumIntKey(account string, scope []byte, denom string, dayUnix int64) []byte {
	legacy := SpendAccumKey(account, scope, denom, dayUnix)
	key := append([]byte{}, SpendAccumIntPrefix...)
	return append(key, legacy[len(SpendAccumPrefix):]...)
}

// SpendAccumFramedTag is the first byte of a daily-spend key suffix written in
// the framed layout (SpendAccumKeyFramed). A legacy suffix starts with the
// bech32 account, which is printable ASCII, so the two layouts cannot share a
// key under the same prefix.
const SpendAccumFramedTag = 0x00

// SpendAccumKeyFramed builds the daily-spend key for (account, scope, denom,
// UTC day) with every variable-length part length-prefixed (QSR-2026-0433):
//
//	prefix | 0x00 | uvarint(len account) | account | uvarint(len scope) | scope |
//	uvarint(len denom) | denom | day(8)
//
// The legacy layout separated the parts with '/', which scope and denom may
// themselves contain, so distinct (scope, denom) pairs could share a bucket
// (scope "/abc" with denom "uqor" is the nil scope with denom "abc/uqor").
// It is written from the abstractaccount-hardening protocol feature on. The
// prefix and the trailing day are those of the legacy layout, so export,
// import and pruning treat both alike.
func SpendAccumKeyFramed(account string, scope []byte, denom string, dayUnix int64) []byte {
	return spendAccumFramed(SpendAccumPrefix, account, scope, denom, dayUnix)
}

// SpendAccumIntKeyFramed is SpendAccumKeyFramed under SpendAccumIntPrefix.
func SpendAccumIntKeyFramed(account string, scope []byte, denom string, dayUnix int64) []byte {
	return spendAccumFramed(SpendAccumIntPrefix, account, scope, denom, dayUnix)
}

func spendAccumFramed(pfx []byte, account string, scope []byte, denom string, dayUnix int64) []byte {
	key := make([]byte, 0, len(pfx)+1+3*binary.MaxVarintLen64+len(account)+len(scope)+len(denom)+8)
	key = append(key, pfx...)
	key = append(key, SpendAccumFramedTag)
	key = binary.AppendUvarint(key, uint64(len(account)))
	key = append(key, account...)
	key = binary.AppendUvarint(key, uint64(len(scope)))
	key = append(key, scope...)
	key = binary.AppendUvarint(key, uint64(len(denom)))
	key = append(key, denom...)
	var d [8]byte
	binary.BigEndian.PutUint64(d[:], uint64(dayUnix))
	return append(key, d[:]...)
}

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
	// A nil scope is the account-wide bucket. It was meant not to collide with a
	// key-scoped one because a public key is never empty, but a scope starting
	// with '/' (0x2f) does collide with it; SpendAccumKeyFramed is the fix.
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
