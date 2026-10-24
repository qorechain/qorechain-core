package types

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const testToken = "0x00000000000000000000000000000000000000Aa"

func TestDenomLimitBoundsNothing(t *testing.T) {
	cases := []struct {
		dl   *DenomLimit
		want bool
	}{
		{&DenomLimit{Denom: "uqor"}, true},
		{&DenomLimit{Denom: "uqor", PerTxLimit: "0", DailyLimit: ""}, true},
		{&DenomLimit{Denom: "uqor", PerTxLimit: "0", DailyLimit: "0"}, true},
		{&DenomLimit{Denom: "uqor", PerTxLimit: "1"}, false},
		{&DenomLimit{Denom: "uqor", DailyLimit: "5"}, false},
		{&DenomLimit{Denom: "uqor", PerTxLimit: "x"}, false}, // malformed: refused elsewhere
		{nil, false},
	}
	for i, c := range cases {
		if got := DenomLimitBoundsNothing(c.dl); got != c.want {
			t.Fatalf("case %d: got %v want %v", i, got, c.want)
		}
	}
}

func TestValidateEVMDenom(t *testing.T) {
	ok := []string{
		"uqor", "ibc/ABC", "factory/x/y",
		"erc20/" + testToken, "ERC20/" + strings.ToLower(testToken),
		"erc721/" + testToken, "erc1155/" + testToken + "/7",
	}
	for _, d := range ok {
		if err := ValidateEVMDenom(d); err != nil {
			t.Fatalf("%s: %v", d, err)
		}
	}
	bad := []string{
		"erc20/notanaddress", "erc20/", "erc20/00000000000000000000000000000000000000Aa",
		"erc20/0x1234", "erc721/0xZZ00000000000000000000000000000000000000",
		"erc1155/garbage/1", "erc20/" + testToken + "/1",
	}
	for _, d := range bad {
		if err := ValidateEVMDenom(d); err == nil {
			t.Fatalf("%s: accepted", d)
		}
	}
}

func TestValidateSpendingRuleBounds(t *testing.T) {
	ok := []SpendingRule{
		{Enabled: true, PerTxLimit: 1},
		{Enabled: true, DailyLimit: 1},
		{Enabled: true, AllowedDenoms: []string{"uqor"}},
		{Enabled: true, AllowedEvmCalls: []string{testToken}},
		{Enabled: true, DenomLimits: []*DenomLimit{{Denom: "uqor", PerTxLimit: "10"}}},
		{Enabled: true, DenomLimits: []*DenomLimit{{Denom: "erc20/" + testToken, DailyLimit: "1000000000000000000000"}}},
		{Enabled: false}, // a disabled rule may be stored empty
	}
	for i, r := range ok {
		if err := ValidateSpendingRuleBounds(r); err != nil {
			t.Fatalf("ok %d: %v", i, err)
		}
	}
	bad := []struct {
		rule SpendingRule
		want string
	}{
		// QSR-2026-0611 as reported: enabled, only a vacuous denom_limits entry.
		{SpendingRule{Enabled: true, DenomLimits: []*DenomLimit{{Denom: "uqor"}}}, "neither per_tx_limit nor daily_limit"},
		// The worse case: real rule-level limits plus an empty entry for uqor,
		// which would make uqor unbounded.
		{SpendingRule{Enabled: true, PerTxLimit: 100, DailyLimit: 1000, DenomLimits: []*DenomLimit{{Denom: "uqor", PerTxLimit: "0", DailyLimit: "0"}}}, "neither per_tx_limit nor daily_limit"},
		// Refused on a disabled rule too: enabling it later must not bring the hole back.
		{SpendingRule{Enabled: false, DenomLimits: []*DenomLimit{{Denom: "uqor"}}}, "neither per_tx_limit nor daily_limit"},
		{SpendingRule{Enabled: true, DenomLimits: []*DenomLimit{{Denom: "erc20/notanaddress", PerTxLimit: "1"}}}, "contract address"},
		{SpendingRule{Enabled: true, AllowedDenoms: []string{"erc721/nope"}}, "contract address"},
		{SpendingRule{Enabled: true}, "enabled but sets no"},
	}
	for i, c := range bad {
		err := ValidateSpendingRuleBounds(c.rule)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("bad %d: got %v, want %q", i, err, c.want)
		}
	}
}

// The new checks are NOT part of ValidateBasic: transactions already in the
// testnet's history carry rules that the save-time check now refuses, and
// they must still pass the stateless check on replay.
func TestValidateBasicUnchangedByRuleBounds(t *testing.T) {
	owner := sdk.AccAddress(make([]byte, 20)).String()
	rules := []SpendingRule{
		{Enabled: true, PerTxLimit: 100, DenomLimits: []*DenomLimit{{Denom: "uqor"}}},
		{Enabled: true, DenomLimits: []*DenomLimit{{Denom: "erc20/notanaddress"}}},
	}
	for i, r := range rules {
		msg := MsgUpdateSpendingRules{Owner: owner, AccountAddress: owner, Rules: []SpendingRule{r}}
		if err := msg.ValidateBasic(); err != nil {
			t.Fatalf("rule %d: ValidateBasic now refuses a historical shape: %v", i, err)
		}
		if err := ValidateSpendingRuleBounds(r); err == nil {
			t.Fatalf("rule %d: save-time check accepts it", i)
		}
	}
}

// Duplicates are compared by canonical denom (QSR-2026-0611 item 3), already
// refused by ValidateSpendingPolicy.
func TestDuplicateDenomLimitsByCanonicalDenom(t *testing.T) {
	r := SpendingRule{Enabled: true, DenomLimits: []*DenomLimit{
		{Denom: "erc20/" + testToken, PerTxLimit: "1"},
		{Denom: "ERC20/" + strings.ToLower(testToken), PerTxLimit: "2"},
	}}
	if err := ValidateSpendingPolicy(r); err == nil || !strings.Contains(err.Error(), "more than one limit") {
		t.Fatalf("got %v", err)
	}
}
