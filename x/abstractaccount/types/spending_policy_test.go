package types

import (
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const tokLower = "0x00000000000000000000000000000000000a11ce"

func TestCanonicalDenom(t *testing.T) {
	cs := common.HexToAddress(tokLower).Hex()
	want := "erc20/" + cs
	for _, in := range []string{"erc20/" + tokLower, "ERC20/" + strings.ToUpper(tokLower[:2]) + strings.ToUpper(tokLower[2:]), want} {
		if got := CanonicalDenom(in); got != want {
			t.Fatalf("CanonicalDenom(%q) = %q, want %q", in, got, want)
		}
	}
	if got := CanonicalDenom("erc1155/" + tokLower + "/7"); got != "erc1155/"+cs+"/7" {
		t.Fatalf("erc1155 id kept: %q", got)
	}
	for _, same := range []string{"uqor", "ibc/ABC", "erc20/notanaddress", "erc20/00000000000000000000000000000000000a11ce"} {
		if got := CanonicalDenom(same); got != same {
			t.Fatalf("%q changed to %q", same, got)
		}
	}
	if a, ok := ListedERC20("erc20/" + tokLower); !ok || strings.ToLower(a.Hex()) != tokLower {
		t.Fatalf("ListedERC20: %s %v", a.Hex(), ok)
	}
	if _, ok := ListedERC20("uqor"); ok {
		t.Fatal("uqor is not an ERC-20")
	}
}

func TestParseAllowedEVMCall(t *testing.T) {
	c, err := ParseAllowedEVMCall(tokLower + "/0x38ed1739")
	if err != nil || !c.HasSelector || c.Selector != [4]byte{0x38, 0xed, 0x17, 0x39} {
		t.Fatalf("target/selector: %+v %v", c, err)
	}
	c, err = ParseAllowedEVMCall(tokLower)
	if err != nil || c.HasSelector {
		t.Fatalf("bare target: %+v %v", c, err)
	}
	for _, bad := range []string{"", "0x12", tokLower + "/38ed1739", tokLower + "/0x38ed17", tokLower + "/0xzzzzzzzz", tokLower[2:]} {
		if _, err := ParseAllowedEVMCall(bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
}

func TestValidateSpendingPolicy(t *testing.T) {
	ok := SpendingRule{
		Enabled:         true,
		AllowedEvmCalls: []string{tokLower},
		DenomLimits:     []*DenomLimit{{Denom: "erc20/" + tokLower, PerTxLimit: "1000000000000000000000000", DailyLimit: ""}},
	}
	if err := ValidateSpendingPolicy(ok); err != nil {
		t.Fatal(err)
	}
	dup := ok
	dup.DenomLimits = append(dup.DenomLimits, &DenomLimit{Denom: "ERC20/" + tokLower, DailyLimit: "1"})
	if err := ValidateSpendingPolicy(dup); err == nil {
		t.Fatal("case-variant duplicate denom accepted")
	}
	for _, dl := range []*DenomLimit{nil, {Denom: ""}, {Denom: "uqor", PerTxLimit: "-1"}, {Denom: "uqor", DailyLimit: "1.5"}, {Denom: "u qor"}} {
		r := SpendingRule{DenomLimits: []*DenomLimit{dl}}
		if err := ValidateSpendingPolicy(r); err == nil {
			t.Fatalf("bad denom limit %+v accepted", dl)
		}
	}
	many := SpendingRule{}
	for i := 0; i <= MaxListedERC20; i++ {
		many.AllowedDenoms = append(many.AllowedDenoms, "erc20/0x"+strings.Repeat("0", 38)+string("0123456789abcdef"[i%16])+string("0123456789abcdef"[i/16]))
	}
	if err := ValidateSpendingPolicy(many); err == nil {
		t.Fatal("too many listed tokens accepted")
	}
}

func TestMsgUpdateSpendingRulesValidateBasicNewFields(t *testing.T) {
	owner := sdk.AccAddress(make([]byte, 20)).String()
	base := MsgUpdateSpendingRules{Owner: owner, AccountAddress: owner}

	// A rule bounded only by the new fields is a bound.
	m := base
	m.Rules = []SpendingRule{{Enabled: true, AllowedEvmCalls: []string{tokLower}}}
	if err := m.ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	m.Rules = []SpendingRule{{Enabled: true, DenomLimits: []*DenomLimit{{Denom: "uqor", DailyLimit: "5"}}}}
	if err := m.ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	// A malformed new field is refused before it reaches the chain.
	m.Rules = []SpendingRule{{Enabled: true, AllowedEvmCalls: []string{"nope"}}}
	if err := m.ValidateBasic(); err == nil {
		t.Fatal("malformed allowed_evm_calls accepted")
	}
	// A legacy rule is validated exactly as before: an enabled empty rule is
	// refused, and a legacy rule listing many tokens is not the new check's
	// business.
	m.Rules = []SpendingRule{{Enabled: true}}
	if err := m.ValidateBasic(); err == nil {
		t.Fatal("empty enabled rule accepted")
	}
}

// The JSON the keeper stores for a rule without the new fields must be exactly
// what v3.2.0 stored: account records are rewritten below the plan height too.
func TestSpendingRuleJSONOmitsNewFields(t *testing.T) {
	bz, err := json.Marshal(SpendingRule{ID: "r", DailyLimit: 1, PerTxLimit: 2, AllowedDenoms: []string{"uqor"}, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(bz) != `{"id":"r","daily_limit":1,"per_tx_limit":2,"allowed_denoms":["uqor"],"enabled":true}` {
		t.Fatalf("legacy JSON changed: %s", bz)
	}
	var back SpendingRule
	if err := json.Unmarshal([]byte(`{"id":"r","daily_limit":1,"allowed_evm_calls":["`+tokLower+`"],"denom_limits":[{"denom":"uqor","daily_limit":"9"}]}`), &back); err != nil {
		t.Fatal(err)
	}
	if len(back.AllowedEvmCalls) != 1 || len(back.DenomLimits) != 1 || back.DenomLimits[0].DailyLimit != "9" {
		t.Fatalf("new fields did not round-trip: %+v", back)
	}
}

func TestGenesisValidateIntBuckets(t *testing.T) {
	acct := sdk.AccAddress(make([]byte, 20)).String()
	suffix := SpendAccumIntKey(acct, nil, "erc20/"+tokLower, 20000)[len(SpendAccumIntPrefix):]
	gs := DefaultGenesisState()
	gs.SpendAccumulatorsInt = []SpendAccumIntEntry{{KeySuffix: suffix, Amount: "115792089237316195423570985008687907853269984665640564039457584007913129639935"}}
	if err := gs.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, amt := range []string{"", "0", "-1", "x"} {
		gs.SpendAccumulatorsInt[0].Amount = amt
		if err := gs.Validate(); err == nil {
			t.Fatalf("amount %q accepted", amt)
		}
	}
	gs.SpendAccumulatorsInt = []SpendAccumIntEntry{{KeySuffix: suffix, Amount: "1"}, {KeySuffix: suffix, Amount: "2"}}
	if err := gs.Validate(); err == nil {
		t.Fatal("duplicate bucket accepted")
	}
	// The default genesis has no such field on the wire.
	bz, _ := json.Marshal(DefaultGenesisState())
	if strings.Contains(string(bz), "spend_accumulators_int") {
		t.Fatalf("default genesis changed: %s", bz)
	}
}

func TestLaneRulesNameDeployForCreation(t *testing.T) {
	rules := LaneRules()
	if len(rules) == 0 || rules[0].MsgTypeUrl != MsgExecuteEVMTypeURL || rules[0].Permission != PermDeploy {
		t.Fatalf("lane rules: %+v", rules)
	}
	if PermSchemaVersionLaneRules == PermSchemaVersion {
		t.Fatal("lane rules must bump the schema version")
	}
}
