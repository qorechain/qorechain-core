package types

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRollupModeMatrix(t *testing.T) {
	opt := RollupConfig{SettlementMode: SettlementOptimistic, ProofConfig: ProofConfig{System: ProofSystemFraud}}
	zk := RollupConfig{SettlementMode: SettlementZK, ProofConfig: ProofConfig{System: ProofSystemSNARK}}
	starkCfg := RollupConfig{SettlementMode: SettlementZK, ProofConfig: ProofConfig{System: ProofSystemSTARK}}
	member := sdk.AccAddress([]byte("rdk-types-member-001")).String()
	cases := []struct {
		name string
		mod  func(RollupConfig) RollupConfig
		base RollupConfig
		ok   bool
	}{
		{"qledger based", func(c RollupConfig) RollupConfig {
			c.ExecutionVM, c.Sequencing = ExecutionVMQLedgerV1, SequencingBased
			return c
		}, opt, true},
		{"qledger not based", func(c RollupConfig) RollupConfig { c.ExecutionVM = ExecutionVMQLedgerV1; return c }, opt, false},
		{"qledger on zk", func(c RollupConfig) RollupConfig {
			c.ExecutionVM, c.Sequencing = ExecutionVMQLedgerV1, SequencingBased
			return c
		}, zk, false},
		{"unknown vm", func(c RollupConfig) RollupConfig { c.ExecutionVM = "wasm"; return c }, opt, false},
		{"private snark", func(c RollupConfig) RollupConfig { c.Privacy = PrivacyEncapsulated; return c }, zk, true},
		{"private snark with members", func(c RollupConfig) RollupConfig {
			c.Privacy, c.Members = PrivacyEncapsulated, []string{member}
			return c
		}, zk, true},
		{"private optimistic", func(c RollupConfig) RollupConfig { c.Privacy = PrivacyEncapsulated; return c }, opt, false},
		{"private stark", func(c RollupConfig) RollupConfig { c.Privacy = PrivacyEncapsulated; return c }, starkCfg, false},
		{"members without privacy", func(c RollupConfig) RollupConfig { c.Members = []string{member}; return c }, zk, false},
		{"duplicate member", func(c RollupConfig) RollupConfig {
			c.Privacy, c.Members = PrivacyEncapsulated, []string{member, member}
			return c
		}, zk, false},
	}
	for _, c := range cases {
		err := c.mod(c.base).validateModes()
		if c.ok && err == nil {
			if c.mod(c.base).IsBased() {
				if berr := c.mod(c.base).BasedEligible(); berr != nil {
					t.Errorf("%s: based eligibility: %v", c.name, berr)
				}
			}
			continue
		}
		if c.ok != (err == nil) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
	// A plain optimistic rollup still cannot be based.
	if err := (RollupConfig{SettlementMode: SettlementOptimistic, Sequencing: SequencingBased}).BasedEligible(); !errors.Is(err, ErrBasedSequencingUnsupported) {
		t.Fatalf("plain optimistic based: %v", err)
	}
}

func TestModeMsgsValidateBasic(t *testing.T) {
	a := sdk.AccAddress([]byte("rdk-types-sender-001")).String()
	acct := make([]byte, 32)
	acct[0] = 1
	if err := (&MsgDepositToRollup{Depositor: a, RollupID: "r", Account: acct, Amount: MinDepositAmount}).ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	for name, m := range map[string]*MsgDepositToRollup{
		"short account": {Depositor: a, RollupID: "r", Account: acct[:31], Amount: MinDepositAmount},
		"zero account":  {Depositor: a, RollupID: "r", Account: make([]byte, 32), Amount: MinDepositAmount},
		"dust":          {Depositor: a, RollupID: "r", Account: acct, Amount: MinDepositAmount - 1},
		"no rollup":     {Depositor: a, Account: acct, Amount: MinDepositAmount},
	} {
		if m.ValidateBasic() == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if (&MsgBondRollupSubmitter{Submitter: a, RollupID: "r"}).ValidateBasic() == nil {
		t.Error("zero bond accepted")
	}
	many := make([]string, MaxRollupMembers+1)
	for i := range many {
		many[i] = sdk.AccAddress(append([]byte("rdk-types-member-"), byte(i), 0, 0)).String()
	}
	if (&MsgSetRollupMembers{Creator: a, RollupID: "r", Members: many}).ValidateBasic() == nil {
		t.Error("too many members accepted")
	}
	if (&MsgSetRollupMembers{Creator: a, RollupID: "r"}).ValidateBasic() == nil {
		t.Error("empty member list accepted")
	}
	if (&MsgSetRollupMembers{Creator: a, RollupID: "r", Members: many[:2]}).ValidateBasic() != nil {
		t.Error("valid member list refused")
	}
	p := DefaultParams()
	p.MinSubmitterBond = MaxMinSubmitterBond + 1
	if p.validateModes() == nil {
		t.Error("oversized min_submitter_bond accepted")
	}
	if (Params{}).SubmitterBondMin() != DefaultMinSubmitterBond {
		t.Error("default bond")
	}
}
