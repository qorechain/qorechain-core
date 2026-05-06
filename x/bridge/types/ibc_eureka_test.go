package types

import "testing"

func goodEurekaPacket() EurekaPacket {
	return EurekaPacket{
		SourceChain:      "cosmoshub",
		DestChain:        "qorechain-diana",
		Sequence:         1,
		PortID:           "transfer",
		ChannelID:        "channel-0",
		ClientType:       "tendermint",
		Data:             []byte(`{"denom":"uatom","amount":"1000000","sender":"cosmos1...","receiver":"qor1..."}`),
		TimeoutTimestamp: 0,
	}
}

func TestEurekaPacket_Validate_OK(t *testing.T) {
	if err := goodEurekaPacket().Validate(); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestEurekaPacket_Validate_RejectsEmptyFields(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*EurekaPacket)
		wantSub string
	}{
		{"empty_source", func(p *EurekaPacket) { p.SourceChain = "" }, "source_chain"},
		{"empty_dest", func(p *EurekaPacket) { p.DestChain = "" }, "dest_chain"},
		{"empty_port", func(p *EurekaPacket) { p.PortID = "" }, "port_id"},
		{"empty_channel", func(p *EurekaPacket) { p.ChannelID = "" }, "channel_id"},
		{"empty_client_type", func(p *EurekaPacket) { p.ClientType = "" }, "client_type"},
		{"zero_sequence", func(p *EurekaPacket) { p.Sequence = 0 }, "sequence"},
		{"empty_data", func(p *EurekaPacket) { p.Data = nil }, "data"},
		{"same_source_dest", func(p *EurekaPacket) { p.DestChain = p.SourceChain }, "must differ"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := goodEurekaPacket()
			c.mutate(&p)
			err := p.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !contains(err.Error(), c.wantSub) {
				t.Errorf("error %q does not mention %q", err.Error(), c.wantSub)
			}
		})
	}
}

func TestEurekaAck_IsSuccess(t *testing.T) {
	cases := []struct {
		ack  EurekaAck
		want bool
	}{
		{EurekaAck{Success: true}, true},
		{EurekaAck{Success: false}, false},
		{EurekaAck{Success: true, Error: "boom"}, false}, // success+error is contradictory; reject
		{EurekaAck{Success: false, Error: "channel closed"}, false},
	}
	for _, c := range cases {
		if got := c.ack.IsSuccess(); got != c.want {
			t.Errorf("EurekaAck{Success:%v, Error:%q}.IsSuccess() = %v, want %v",
				c.ack.Success, c.ack.Error, got, c.want)
		}
	}
}

func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestEurekaHandlerHook_InterfaceShape — compile-time check that any
// real implementation will have the expected method set. This guards
// against accidental signature drift when the proprietary keeper's
// adapter is updated.
func TestEurekaHandlerHook_InterfaceShape(t *testing.T) {
	var _ EurekaHandlerHook = (*fakeEurekaHandler)(nil)
}

type fakeEurekaHandler struct{}

func (fakeEurekaHandler) OnRecvPacket(_ EurekaPacket) EurekaAck { return EurekaAck{Success: true} }
func (fakeEurekaHandler) OnAcknowledgement(_ EurekaPacket, _ EurekaAck) error { return nil }
func (fakeEurekaHandler) OnTimeout(_ EurekaPacket) error { return nil }
