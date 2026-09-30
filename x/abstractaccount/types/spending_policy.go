package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
)

// v3.2.2 spending policy (QSR-2026-0037, 0088, 0184, 0226): the parts of a
// SpendingRule that clients author and the chain validates. Enforcement lives in
// the keeper.

// Bounds on what one rule may list. Every listed ERC-20 is probed on every EVM
// call of a key bound by the rule, and the relayer pays for that, so the lists
// stay short.
const (
	MaxAllowedEVMCalls = 64
	MaxDenomLimits     = 64
	MaxListedERC20     = 16
)

// Denom prefixes that name an EVM contract. The part after the prefix is an EVM
// address, compared in its EIP-55 form.
var evmDenomPrefixes = []string{"erc20/", "erc721/", "erc1155/"}

// CanonicalDenom returns the form of a denom that the spending policy compares
// and counts under: an "erc20/", "erc721/" or "erc1155/" denom gets a lower-case
// prefix and the EIP-55 checksum form of its contract address (anything after
// the address, such as an ERC-1155 token id, is kept as written). Any other
// denom is returned unchanged. Without this, "erc20/0xabc..." and
// "erc20/0xABC..." would be different denoms to the allow list and separate
// daily buckets for the same token.
func CanonicalDenom(denom string) string {
	lower := strings.ToLower(denom)
	for _, p := range evmDenomPrefixes {
		if !strings.HasPrefix(lower, p) {
			continue
		}
		rest := denom[len(p):]
		addr, tail := rest, ""
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			addr, tail = rest[:i], rest[i:]
		}
		if !common.IsHexAddress(addr) || !strings.HasPrefix(strings.ToLower(addr), "0x") {
			return denom
		}
		return p + common.HexToAddress(addr).Hex() + tail
	}
	return denom
}

// ERC20Denom is the canonical denom of an ERC-20 contract.
func ERC20Denom(token common.Address) string { return "erc20/" + token.Hex() }

// ListedERC20 returns the contract address of a canonical "erc20/<address>"
// denom.
func ListedERC20(denom string) (common.Address, bool) {
	d := CanonicalDenom(denom)
	if !strings.HasPrefix(d, "erc20/") {
		return common.Address{}, false
	}
	a := d[len("erc20/"):]
	if !common.IsHexAddress(a) {
		return common.Address{}, false
	}
	return common.HexToAddress(a), true
}

// AllowedEVMCall is one parsed SpendingRule.AllowedEvmCalls entry.
type AllowedEVMCall struct {
	Target      common.Address
	Selector    [4]byte
	HasSelector bool
}

// ParseAllowedEVMCall parses "0x<target>" or "0x<target>/0x<selector>".
func ParseAllowedEVMCall(entry string) (AllowedEVMCall, error) {
	target, sel, hasSel := entry, "", false
	if i := strings.IndexByte(entry, '/'); i >= 0 {
		target, sel, hasSel = entry[:i], entry[i+1:], true
	}
	if !strings.HasPrefix(strings.ToLower(target), "0x") || !common.IsHexAddress(target) {
		return AllowedEVMCall{}, fmt.Errorf("allowed EVM call %q: target must be a 0x-prefixed 20-byte address", entry)
	}
	out := AllowedEVMCall{Target: common.HexToAddress(target)}
	if hasSel {
		if !strings.HasPrefix(strings.ToLower(sel), "0x") || len(sel) != 10 {
			return AllowedEVMCall{}, fmt.Errorf("allowed EVM call %q: selector must be 0x followed by 8 hex digits", entry)
		}
		b, err := hex.DecodeString(sel[2:])
		if err != nil {
			return AllowedEVMCall{}, fmt.Errorf("allowed EVM call %q: selector: %w", entry, err)
		}
		copy(out.Selector[:], b)
		out.HasSelector = true
	}
	return out, nil
}

// ParseLimitAmount parses a DenomLimit amount: a non-negative decimal integer,
// with "" meaning zero (no limit of that kind).
func ParseLimitAmount(s string) (sdkmath.Int, error) {
	if s == "" {
		return sdkmath.ZeroInt(), nil
	}
	v, ok := sdkmath.NewIntFromString(s)
	if !ok || v.IsNegative() {
		return sdkmath.Int{}, fmt.Errorf("limit %q is not a non-negative decimal integer", s)
	}
	return v, nil
}

// ValidateSpendingPolicy checks the v3.2.2 parts of a rule: the allowed EVM
// calls, the per-denom limits and the number of listed ERC-20 tokens. A rule
// that uses none of them passes unchanged.
func ValidateSpendingPolicy(rule SpendingRule) error {
	if len(rule.AllowedEvmCalls) > MaxAllowedEVMCalls {
		return fmt.Errorf("at most %d allowed EVM calls per rule, got %d", MaxAllowedEVMCalls, len(rule.AllowedEvmCalls))
	}
	for _, e := range rule.AllowedEvmCalls {
		if _, err := ParseAllowedEVMCall(e); err != nil {
			return err
		}
	}
	if len(rule.DenomLimits) > MaxDenomLimits {
		return fmt.Errorf("at most %d denom limits per rule, got %d", MaxDenomLimits, len(rule.DenomLimits))
	}
	seen := make(map[string]struct{}, len(rule.DenomLimits))
	for _, dl := range rule.DenomLimits {
		if dl == nil || strings.TrimSpace(dl.Denom) == "" {
			return fmt.Errorf("denom limit without a denom")
		}
		if err := sdkValidateDenomOrEVM(dl.Denom); err != nil {
			return err
		}
		c := CanonicalDenom(dl.Denom)
		if _, dup := seen[c]; dup {
			return fmt.Errorf("denom %s has more than one limit", c)
		}
		seen[c] = struct{}{}
		if _, err := ParseLimitAmount(dl.PerTxLimit); err != nil {
			return fmt.Errorf("denom %s per-tx: %w", c, err)
		}
		if _, err := ParseLimitAmount(dl.DailyLimit); err != nil {
			return fmt.Errorf("denom %s daily: %w", c, err)
		}
	}
	erc20 := 0
	for _, d := range rule.AllowedDenoms {
		if _, ok := ListedERC20(d); ok {
			erc20++
		}
	}
	if erc20 > MaxListedERC20 {
		return fmt.Errorf("at most %d ERC-20 tokens per rule, got %d", MaxListedERC20, erc20)
	}
	return nil
}

// sdkValidateDenomOrEVM accepts any denom without whitespace or control bytes;
// the bank's own denom grammar is not imposed because EVM denoms carry '/' and
// mixed case.
func sdkValidateDenomOrEVM(d string) error {
	if len(d) > 256 {
		return fmt.Errorf("denom %q is too long", d)
	}
	for _, r := range d {
		if r <= ' ' || r == 0x7f {
			return fmt.Errorf("denom %q contains whitespace or control characters", d)
		}
	}
	return nil
}

// UsesSpendingPolicyFields reports whether a rule carries any field added in
// v3.2.2 (allowed_evm_calls, denom_limits).
func UsesSpendingPolicyFields(rule SpendingRule) bool {
	return len(rule.AllowedEvmCalls) > 0 || len(rule.DenomLimits) > 0
}
