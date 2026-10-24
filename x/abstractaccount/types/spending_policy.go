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

// DenomLimitBoundsNothing reports whether a denom_limits entry sets neither a
// per-tx nor a daily amount ("" and "0" both mean no limit of that kind). Such
// an entry still replaces the rule's per_tx_limit and daily_limit for its
// denom, so under the v3.2.2 check it made that denom unbounded
// (QSR-2026-0611). An entry whose amounts do not parse is not reported here;
// ValidateSpendingPolicy refuses it.
func DenomLimitBoundsNothing(dl *DenomLimit) bool {
	if dl == nil {
		return false
	}
	perTx, err := ParseLimitAmount(dl.PerTxLimit)
	if err != nil {
		return false
	}
	daily, err := ParseLimitAmount(dl.DailyLimit)
	if err != nil {
		return false
	}
	return perTx.IsZero() && daily.IsZero()
}

// ValidateEVMDenom refuses an "erc20/", "erc721/" or "erc1155/" denom (any
// case) whose contract address is not a 0x-prefixed 20-byte address, and an
// "erc20/" denom with anything after the address. Neither can ever match what
// the chain measures, so a rule naming one would list or cap nothing while
// appearing to. Any other denom passes.
func ValidateEVMDenom(denom string) error {
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
		if !strings.HasPrefix(strings.ToLower(addr), "0x") || !common.IsHexAddress(addr) {
			return fmt.Errorf("denom %q: %s must be followed by a 0x-prefixed 20-byte contract address", denom, p)
		}
		if p == "erc20/" && tail != "" {
			return fmt.Errorf("denom %q: an erc20/ denom is the contract address alone", denom)
		}
		return nil
	}
	return nil
}

// RuleBoundsSomething reports whether a rule carries at least one real bound:
// a per-tx or daily limit, a denom list, an allowed EVM call list, or a
// denom_limits entry with a positive amount.
func RuleBoundsSomething(rule SpendingRule) bool {
	if rule.PerTxLimit > 0 || rule.DailyLimit > 0 || len(rule.AllowedDenoms) > 0 || len(rule.AllowedEvmCalls) > 0 {
		return true
	}
	for _, dl := range rule.DenomLimits {
		if dl != nil && !DenomLimitBoundsNothing(dl) {
			return true
		}
	}
	return false
}

// ValidateSpendingRuleBounds is the save-time check of QSR-2026-0611, run by
// the message server under FeatureAASpendingRuleBounds and never in
// ValidateBasic, so transactions already in the chain's history replay as
// they did. It refuses:
//   - an enabled rule that bounds nothing (see RuleBoundsSomething);
//   - a denom_limits entry that sets neither amount, because the entry
//     replaces the rule-level limits for its denom and would make that denom
//     unbounded;
//   - an EVM denom whose contract address does not parse (ValidateEVMDenom),
//     in denom_limits or allowed_denoms.
//
// Duplicate denom_limits entries, compared by CanonicalDenom, are refused by
// ValidateSpendingPolicy, which runs first.
func ValidateSpendingRuleBounds(rule SpendingRule) error {
	for _, dl := range rule.DenomLimits {
		if dl == nil {
			return fmt.Errorf("denom limit without a denom")
		}
		if err := ValidateEVMDenom(dl.Denom); err != nil {
			return err
		}
		if DenomLimitBoundsNothing(dl) {
			return fmt.Errorf(
				"denom %s: the limit sets neither per_tx_limit nor daily_limit; an entry "+
					"replaces the rule's limits for its denom, so this one would leave the "+
					"denom unbounded; set an amount or remove the entry", CanonicalDenom(dl.Denom))
		}
	}
	for _, d := range rule.AllowedDenoms {
		if err := ValidateEVMDenom(d); err != nil {
			return err
		}
	}
	if rule.Enabled && !RuleBoundsSomething(rule) {
		return fmt.Errorf(
			"the rule is enabled but sets no per-tx limit, no daily limit, no denom " +
				"list, no allowed EVM call and no denom limit with an amount, so it " +
				"would permit everything while appearing to be a spending limit; set " +
				"at least one bound or disable it")
	}
	return nil
}

// UsesSpendingPolicyFields reports whether a rule carries any field added in
// v3.2.2 (allowed_evm_calls, denom_limits).
func UsesSpendingPolicyFields(rule SpendingRule) bool {
	return len(rule.AllowedEvmCalls) > 0 || len(rule.DenomLimits) > 0
}
