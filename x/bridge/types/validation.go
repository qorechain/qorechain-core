package types

import (
	"bytes"
	"fmt"

	sdkmath "cosmossdk.io/math"
)

// Bridge configuration has several entry points: the genesis file, the admin
// messages (UpdateChainConfig, SetVerifierBootstrap, UpdateCircuitBreakerLimits)
// and the stored records the handlers read back. They used to check different
// things, or nothing, so a value one of them refused could still arrive through
// another (QSR-2026-0016, QSR-2026-0047). The rules live here, once, and every
// entry point calls the same function.

// ParseStoredAmount parses an amount the module keeps in state: a running
// total, a daily counter, or a limit where zero means unlimited. Zero is a
// legitimate value for all of them, so only an unparseable or negative value is
// an error.
//
// ParseAmount stays the parser for amounts that arrive in a message, where zero
// or less has no meaning. Using it on stored fields is what made a fresh route
// fail on its first deposit and on every withdrawal: the counters start at "0".
func ParseStoredAmount(s string) (sdkmath.Int, error) {
	i, ok := sdkmath.NewIntFromString(s)
	if !ok {
		return sdkmath.Int{}, ErrInvalidAmount.Wrapf("cannot parse stored amount: %q", s)
	}
	if i.IsNegative() {
		return sdkmath.Int{}, ErrInvalidAmount.Wrapf("stored amount must not be negative, got %s", s)
	}
	return i, nil
}

// IsValid reports whether s is one of the three statuses, spelled exactly as
// the constants are. "PAUSED" or "Paused" is not a status: stored verbatim it
// used to leave a route open, because the handlers compared against "paused".
func (s BridgeStatus) IsValid() bool {
	switch s {
	case BridgeStatusActive, BridgeStatusPaused, BridgeStatusPending:
		return true
	}
	return false
}

// MaxBridgeIdentifierLength bounds chain ids and asset symbols.
const MaxBridgeIdentifierLength = 64

// validIdentifier accepts [A-Za-z0-9_.-], 1..MaxBridgeIdentifierLength bytes,
// optionally lower case only. The rule that matters is the absence of '/':
// wrapped denoms are "bridge/<chain>/<asset>" and the locked-amount key is
// "<chain>/<asset>", so a separator inside either part makes two different
// pairs share a denom and a ledger entry.
func validIdentifier(kind, s string, lowerOnly bool) error {
	if s == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	if len(s) > MaxBridgeIdentifierLength {
		return fmt.Errorf("%s %q is longer than %d bytes", kind, s, MaxBridgeIdentifierLength)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_', c == '-', c == '.':
		case !lowerOnly && c >= 'A' && c <= 'Z':
		default:
			return fmt.Errorf("%s %q contains %q; allowed are %s", kind, s, c, allowedChars(lowerOnly))
		}
	}
	return nil
}

func allowedChars(lowerOnly bool) string {
	if lowerOnly {
		return "a-z 0-9 _ - ."
	}
	return "A-Z a-z 0-9 _ - ."
}

// ValidateChainID checks a bridge chain id (lower case, no separator).
func ValidateChainID(chainID string) error {
	if err := validIdentifier("chain id", chainID, true); err != nil {
		return ErrChainNotSupported.Wrap(err.Error())
	}
	return nil
}

// ValidateAsset checks an asset symbol (case is kept: "USDC", "ETH").
func ValidateAsset(asset string) error {
	if err := validIdentifier("asset", asset, false); err != nil {
		return ErrAssetNotSupported.Wrap(err.Error())
	}
	return nil
}

// validateLimitPair checks two stored limits where zero means unlimited: both
// must parse as non-negative, and a single transfer may not exceed the daily
// limit when both are set.
func validateLimitPair(maxSingle, daily string) error {
	ms, err := ParseStoredAmount(maxSingle)
	if err != nil {
		return fmt.Errorf("max_single_transfer: %w", err)
	}
	dl, err := ParseStoredAmount(daily)
	if err != nil {
		return fmt.Errorf("daily_limit: %w", err)
	}
	if !ms.IsZero() && !dl.IsZero() && ms.GT(dl) {
		return ErrInvalidAmount.Wrapf("max_single_transfer %s exceeds daily_limit %s", ms, dl)
	}
	return nil
}

// Validate checks a chain configuration as it may be stored.
func (cc ChainConfig) Validate() error {
	if err := ValidateChainID(cc.ChainID); err != nil {
		return err
	}
	if cc.Name == "" {
		return fmt.Errorf("chain config %s has empty name", cc.ChainID)
	}
	if !cc.Status.IsValid() {
		return fmt.Errorf("chain config %s: status %q is not one of active, paused, pending", cc.ChainID, cc.Status)
	}
	if cc.ChainType != "" && !IsValidChainType(cc.ChainType) {
		return fmt.Errorf("chain config %s: unknown chain type %q", cc.ChainID, cc.ChainType)
	}
	if !IsValidChainArchitecture(cc.Architecture) {
		return fmt.Errorf("chain config %s: unknown architecture %q", cc.ChainID, cc.Architecture)
	}
	if cc.MinConfirmations < 0 {
		return fmt.Errorf("chain config %s: min_confirmations must not be negative", cc.ChainID)
	}
	seen := make(map[string]bool, len(cc.SupportedAssets))
	for _, a := range cc.SupportedAssets {
		if err := ValidateAsset(a); err != nil {
			return fmt.Errorf("chain config %s: %w", cc.ChainID, err)
		}
		if seen[a] {
			return fmt.Errorf("chain config %s: asset %s listed twice", cc.ChainID, a)
		}
		seen[a] = true
	}
	if err := validateLimitPair(cc.MaxSingleTransfer, cc.DailyLimit); err != nil {
		return fmt.Errorf("chain config %s: %w", cc.ChainID, err)
	}
	return nil
}

// Validate checks a circuit breaker record. CurrentDaily may be anything
// non-negative: it is a counter, and a limit lowered below today's usage is a
// legitimate way to stop further transfers until the reset.
func (cb CircuitBreakerState) Validate() error {
	if err := ValidateChainID(cb.Chain); err != nil {
		return err
	}
	if err := validateLimitPair(cb.MaxSingleTransfer, cb.DailyLimit); err != nil {
		return fmt.Errorf("circuit breaker %s: %w", cb.Chain, err)
	}
	if _, err := ParseStoredAmount(cb.CurrentDaily); err != nil {
		return fmt.Errorf("circuit breaker %s: current_daily: %w", cb.Chain, err)
	}
	return nil
}

// Validate checks a locked/minted ledger entry: both non-negative, and never
// more minted on this side than is locked on the other.
func (la LockedAmount) Validate() error {
	if err := ValidateChainID(la.Chain); err != nil {
		return err
	}
	if err := ValidateAsset(la.Asset); err != nil {
		return err
	}
	locked, err := ParseStoredAmount(la.TotalLocked)
	if err != nil {
		return fmt.Errorf("locked %s/%s: total_locked: %w", la.Chain, la.Asset, err)
	}
	minted, err := ParseStoredAmount(la.TotalMinted)
	if err != nil {
		return fmt.Errorf("locked %s/%s: total_minted: %w", la.Chain, la.Asset, err)
	}
	if minted.GT(locked) {
		return ErrInvalidAmount.Wrapf("locked %s/%s: total_minted %s exceeds total_locked %s", la.Chain, la.Asset, minted, locked)
	}
	return nil
}

// Key sizes of the verifier trust roots.
const (
	Ed25519PubKeySize     = 32
	BLSG2PubKeySize       = 96 // compressed G2, as GetBLSCommittee decodes it
	WormholeGuardianSize  = 20 // Ethereum-style guardian address
	maxVerifierSetMembers = 256
)

// validateKeySet checks a list of fixed-size keys and its threshold: every key
// the right length, no key twice, and 1 <= threshold <= n.
func validateKeySet(kind string, keys [][]byte, size, threshold int) error {
	n := len(keys)
	if n == 0 {
		return fmt.Errorf("%s set is empty", kind)
	}
	if n > maxVerifierSetMembers {
		return fmt.Errorf("%s set has %d members, max %d", kind, n, maxVerifierSetMembers)
	}
	for i, k := range keys {
		if len(k) != size {
			return fmt.Errorf("%s key %d is %d bytes, want %d", kind, i, len(k), size)
		}
		for j := 0; j < i; j++ {
			if bytes.Equal(keys[j], k) {
				return fmt.Errorf("%s key %d duplicates key %d", kind, i, j)
			}
		}
	}
	if threshold < 1 || threshold > n {
		return fmt.Errorf("%s threshold %d must be between 1 and %d", kind, threshold, n)
	}
	return nil
}

// ValidateEd25519Set checks an ed25519 validator set before it is stored. A key
// of the wrong length would otherwise reach ed25519.Verify, which panics on it.
func ValidateEd25519Set(pubkeys [][]byte, threshold int) error {
	if err := validateKeySet("ed25519", pubkeys, Ed25519PubKeySize, threshold); err != nil {
		return ErrInvalidAttestation.Wrap(err.Error())
	}
	return nil
}

// ValidateBLSCommitteeShape checks a BLS committee's sizes, uniqueness and
// threshold. Whether each key is a point on the curve is checked by the keeper,
// which owns the curve library.
func ValidateBLSCommitteeShape(pubkeys [][]byte, threshold int) error {
	if err := validateKeySet("bls", pubkeys, BLSG2PubKeySize, threshold); err != nil {
		return ErrInvalidAttestation.Wrap(err.Error())
	}
	return nil
}

// WormholeQuorum is the Wormhole rule: more than two thirds of the guardians.
func WormholeQuorum(n int) int { return n*2/3 + 1 }

// ValidateWormholeGuardianSet checks a guardian set: 20-byte addresses, no
// duplicates, and a quorum no lower than the Wormhole rule (and no higher than
// the set).
func ValidateWormholeGuardianSet(addresses [][]byte, quorum int) error {
	if err := validateKeySet("wormhole guardian", addresses, WormholeGuardianSize, quorum); err != nil {
		return ErrInvalidAttestation.Wrap(err.Error())
	}
	if min := WormholeQuorum(len(addresses)); quorum < min {
		return ErrInvalidAttestation.Wrapf("wormhole quorum %d is below the Wormhole rule of %d for %d guardians", quorum, min, len(addresses))
	}
	return nil
}
