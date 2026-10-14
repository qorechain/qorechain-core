package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func testAuthority() string {
	return sdk.AccAddress(make([]byte, 20)).String()
}

// QSR-2026-0434: the mode is a uint32 on the wire and is no longer narrowed at
// decode time. ValidateBasic keeps the historical (narrowing) answer, because
// it cannot tell the height; ValidateModeRange sees the value as sent.
func TestAgentModeWireValue(t *testing.T) {
	msg := &MsgSetAgentMode{Authority: testAuthority(), Mode: 257}
	bz, err := msg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	var got MsgSetAgentMode
	if err := got.Unmarshal(bz); err != nil {
		t.Fatal(err)
	}
	if got.Mode != 257 {
		t.Fatalf("decoded mode %d, want 257 (no narrowing at decode)", got.Mode)
	}
	if err := got.ValidateBasic(); err != nil {
		t.Fatalf("ValidateBasic must keep its historical answer: %v", err)
	}
	if got.AgentModeValue() != AgentModeConservative {
		t.Fatalf("narrowed value %d, want 1", got.AgentModeValue())
	}
	if err := got.ValidateModeRange(); err == nil {
		t.Fatal("ValidateModeRange must refuse 257")
	}
	for m := uint32(0); m <= 3; m++ {
		ok := &MsgSetAgentMode{Authority: testAuthority(), Mode: m}
		if err := ok.ValidateModeRange(); err != nil {
			t.Fatalf("mode %d refused: %v", m, err)
		}
	}
}

// QSR-2026-0435: each weight must be in [0, 1].
func TestRewardWeightsBounds(t *testing.T) {
	if err := DefaultRewardWeights().ValidateBounds(); err != nil {
		t.Fatalf("defaults must pass: %v", err)
	}
	bad := RewardWeights{Throughput: "2.0", Finality: "0", Decentralization: "0", MEV: "-1.0", FailedTxs: "0"}
	if err := bad.Validate(); err != nil {
		t.Fatalf("the historical check accepts it (replay): %v", err)
	}
	if err := bad.ValidateBounds(); err == nil {
		t.Fatal("negative weight must be refused")
	}
}

// QSR-2026-0436 (c): genesis-only fields are bounded.
func TestParamsBounds(t *testing.T) {
	if err := DefaultParams().Validate(); err != nil {
		t.Fatalf("defaults (the stored values on every network) must pass: %v", err)
	}
	cases := map[string]func(*Params){
		"block time 0":        func(p *Params) { p.DefaultBlockTimeMs = 0 },
		"block time negative": func(p *Params) { p.DefaultBlockTimeMs = -5 },
		"negative gas price":  func(p *Params) { p.DefaultBaseGasPrice = "-1" },
		"threshold above 1":   func(p *Params) { p.CircuitBreakerThreshold = "1.5" },
		"threshold negative":  func(p *Params) { p.CircuitBreakerThreshold = "-0.1" },
	}
	for name, mutate := range cases {
		p := DefaultParams()
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Fatalf("%s must be refused", name)
		}
	}
}
