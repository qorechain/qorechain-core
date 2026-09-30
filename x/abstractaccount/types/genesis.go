package types

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

// GenesisSchemaVersion is the genesis layout that carries the authenticator
// sequences and the daily-spend accumulators. A file without it (schema 0)
// came from an export that dropped them.
const GenesisSchemaVersion uint32 = 1

// GenesisState defines the abstractaccount module genesis state.
type GenesisState struct {
	// Schema is GenesisSchemaVersion for a file that carries AuthNonces and
	// SpendAccumulators, 0 for one written before they were exported.
	Schema   uint32                `json:"schema"`
	Config   AbstractAccountConfig `json:"config"`
	Accounts []AbstractAccount     `json:"accounts"`
	// AuthNonces is every stored authenticator sequence, including those of
	// revoked or no longer listed keys: a key relinked after an import must
	// resume from its old sequence, not from 0.
	AuthNonces []AuthNonceEntry `json:"auth_nonces"`
	// SpendAccumulators is every daily-spend bucket still able to bound a spend
	// (the export day and the day before it).
	SpendAccumulators []SpendAccumEntry `json:"spend_accumulators"`
	// SpendAccumulatorsInt is the same for the arbitrary-precision buckets of
	// denoms with their own per-denom limit (v3.2.2). Omitted when empty, so a
	// chain that never used per-denom limits exports the same file as before.
	SpendAccumulatorsInt []SpendAccumIntEntry `json:"spend_accumulators_int,omitempty"`
	// AcceptNonceReset lets a schema-0 file with authenticators load anyway.
	// Every authenticator then restarts at sequence 0, so any signature made on
	// the old chain with the same chain-id becomes valid again. Set it only for
	// a deliberate reset.
	AcceptNonceReset bool `json:"accept_nonce_reset,omitempty"`
}

// AuthNonceEntry is one stored authenticator sequence.
type AuthNonceEntry struct {
	Account   string `json:"account"`
	PubKey    []byte `json:"pubkey"`
	NextNonce uint64 `json:"next_nonce"`
}

// SpendAccumEntry is one daily-spend bucket. KeySuffix is the raw store key
// after SpendAccumPrefix, written back verbatim: its scope and denom parts may
// themselves contain '/', so it is never split into fields.
type SpendAccumEntry struct {
	KeySuffix []byte `json:"key_suffix"`
	Amount    uint64 `json:"amount"`
}

// SpendAccumIntEntry is one arbitrary-precision daily-spend bucket, stored
// under SpendAccumIntPrefix. Amount is a decimal integer.
type SpendAccumIntEntry struct {
	KeySuffix []byte `json:"key_suffix"`
	Amount    string `json:"amount"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Schema:            GenesisSchemaVersion,
		Config:            DefaultAbstractAccountConfig(),
		Accounts:          []AbstractAccount{},
		AuthNonces:        []AuthNonceEntry{},
		SpendAccumulators: []SpendAccumEntry{},
	}
}

// SplitAuthNonceKey splits a stored authenticator-sequence key (without
// AuthNoncePrefix) into its account and pubkey. The account is bech32, which
// has no '/', so the first '/' is the separator and the pubkey is the rest.
func SplitAuthNonceKey(key []byte) (string, []byte, error) {
	i := bytes.IndexByte(key, '/')
	if i <= 0 || i == len(key)-1 {
		return "", nil, fmt.Errorf("malformed authenticator sequence key %x", key)
	}
	return string(key[:i]), append([]byte{}, key[i+1:]...), nil
}

// SpendAccumKeyDay returns the UTC day of a daily-spend key suffix (the key
// without SpendAccumPrefix): the trailing eight bytes, as SpendAccumKey writes
// them.
func SpendAccumKeyDay(suffix []byte) int64 {
	var d int64
	for _, b := range suffix[len(suffix)-8:] {
		d = d<<8 | int64(b)
	}
	return d
}

// validateSpendAccumSuffix checks the only parts of a daily-spend key suffix
// that can be read back unambiguously: a bech32 account, then '/', and at the
// end '/' followed by the eight-byte day. Scope and denom sit in between and
// may contain any byte.
func validateSpendAccumSuffix(suffix []byte) error {
	i := bytes.IndexByte(suffix, '/')
	if i <= 0 {
		return fmt.Errorf("no account part")
	}
	if _, _, err := bech32.DecodeAndConvert(string(suffix[:i])); err != nil {
		return fmt.Errorf("account %q: %w", suffix[:i], err)
	}
	// account '/' scope '/' denom '/' day(8): at least two separators and the
	// day after the account's own separator.
	rest := suffix[i+1:]
	if len(rest) < 2+8 || rest[len(rest)-9] != '/' {
		return fmt.Errorf("no scope/denom/day part")
	}
	return nil
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Config.MaxSessionKeys <= 0 {
		return fmt.Errorf("max session keys must be positive")
	}
	if gs.Config.MaxSpendingRules <= 0 {
		return fmt.Errorf("max spending rules must be positive")
	}
	if gs.Config.DefaultSessionTTL <= 0 {
		return fmt.Errorf("default session TTL must be positive")
	}
	if gs.Schema > GenesisSchemaVersion {
		return fmt.Errorf("genesis schema %d is newer than this binary understands (%d)", gs.Schema, GenesisSchemaVersion)
	}

	if gs.Schema == 0 && !gs.AcceptNonceReset {
		for _, acc := range gs.Accounts {
			if len(acc.Authenticators) > 0 {
				return fmt.Errorf("genesis schema 0 has authenticators (account %s) but no authenticator sequences: "+
					"the export that wrote it dropped them, and importing it would make every past authenticator "+
					"signature valid again; re-export with a current binary, or set accept_nonce_reset to reset them deliberately",
					acc.Address)
			}
		}
	}

	seenNonce := make(map[string]struct{}, len(gs.AuthNonces))
	for i, n := range gs.AuthNonces {
		if _, _, err := bech32.DecodeAndConvert(n.Account); err != nil {
			return fmt.Errorf("auth_nonces[%d]: account %q: %w", i, n.Account, err)
		}
		if len(n.PubKey) == 0 {
			return fmt.Errorf("auth_nonces[%d]: empty pubkey", i)
		}
		k := string(AuthNonceKey(n.Account, n.PubKey))
		if _, dup := seenNonce[k]; dup {
			return fmt.Errorf("auth_nonces[%d]: duplicate entry for %s", i, n.Account)
		}
		seenNonce[k] = struct{}{}
	}

	seenSpend := make(map[string]struct{}, len(gs.SpendAccumulators))
	for i, s := range gs.SpendAccumulators {
		if err := validateSpendAccumSuffix(s.KeySuffix); err != nil {
			return fmt.Errorf("spend_accumulators[%d]: %w", i, err)
		}
		if _, dup := seenSpend[string(s.KeySuffix)]; dup {
			return fmt.Errorf("spend_accumulators[%d]: duplicate key", i)
		}
		seenSpend[string(s.KeySuffix)] = struct{}{}
	}

	seenSpendInt := make(map[string]struct{}, len(gs.SpendAccumulatorsInt))
	for i, s := range gs.SpendAccumulatorsInt {
		if err := validateSpendAccumSuffix(s.KeySuffix); err != nil {
			return fmt.Errorf("spend_accumulators_int[%d]: %w", i, err)
		}
		if v, err := ParseLimitAmount(s.Amount); err != nil || s.Amount == "" {
			return fmt.Errorf("spend_accumulators_int[%d]: amount %q is not a non-negative integer", i, s.Amount)
		} else if v.IsZero() {
			return fmt.Errorf("spend_accumulators_int[%d]: zero amount", i)
		}
		if _, dup := seenSpendInt[string(s.KeySuffix)]; dup {
			return fmt.Errorf("spend_accumulators_int[%d]: duplicate key", i)
		}
		seenSpendInt[string(s.KeySuffix)] = struct{}{}
	}
	return nil
}
