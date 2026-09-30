package types

import (
	"encoding/json"
	"testing"
)

// The message carries the params field by field; converting back must give
// exactly the struct the keeper stores, so a proposal that restates today's
// values writes today's bytes.
func TestMsgUpdateParamsRoundTrip(t *testing.T) {
	p := DefaultParams()
	msg := MsgUpdateParams{
		Authority:                  "qor1x",
		MaxSidechains:              p.MaxSidechains,
		MaxPaychains:               p.MaxPaychains,
		MinAnchorInterval:          p.MinAnchorInterval,
		MaxAnchorInterval:          p.MaxAnchorInterval,
		DefaultChallengePeriod:     p.DefaultChallengePeriod,
		MinSidechainStake:          p.MinSidechainStake,
		MinPaychainStake:           p.MinPaychainStake,
		RoutingEnabled:             p.RoutingEnabled,
		RoutingConfidenceThreshold: p.RoutingConfidenceThreshold,
		CrossLayerFeeBundling:      p.CrossLayerFeeBundling,
	}
	a, _ := json.Marshal(msg.ToParams())
	b, _ := json.Marshal(p)
	if string(a) != string(b) {
		t.Fatalf("round trip changed the stored bytes:\n%s\n%s", a, b)
	}
	if err := msg.ToParams().ValidateUpdate(); err != nil {
		t.Fatalf("default params refused: %v", err)
	}
}

func TestValidateUpdate(t *testing.T) {
	for name, mutate := range map[string]func(*Params){
		"negative stake":      func(p *Params) { p.MinSidechainStake = "-1" },
		"stake not a number":  func(p *Params) { p.MinPaychainStake = "ten" },
		"threshold above one": func(p *Params) { p.RoutingConfidenceThreshold = "1.5" },
		"threshold garbage":   func(p *Params) { p.RoutingConfidenceThreshold = "high" },
		"zero sidechains":     func(p *Params) { p.MaxSidechains = 0 },
		"inverted intervals":  func(p *Params) { p.MaxAnchorInterval = p.MinAnchorInterval },
	} {
		p := DefaultParams()
		mutate(&p)
		if p.ValidateUpdate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
