package types

import "testing"

// ----- ICA tests -----

func TestICARegistration_Validate(t *testing.T) {
	good := ICARegistration{Owner: "qor1xxx", ConnectionID: "connection-0"}
	if err := good.Validate(); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
	if err := (ICARegistration{ConnectionID: "connection-0"}).Validate(); err == nil {
		t.Error("expected error for empty owner")
	}
	if err := (ICARegistration{Owner: "qor1xxx"}).Validate(); err == nil {
		t.Error("expected error for empty connection_id")
	}
}

func TestICAHandlerHook_InterfaceShape(t *testing.T) {
	var _ ICAHandlerHook = (*fakeICAHandler)(nil)
}

type fakeICAHandler struct{}

func (fakeICAHandler) RegisterICA(_ ICARegistration) error                             { return nil }
func (fakeICAHandler) QueryICA(_, _ string) (string, bool)                             { return "", false }
func (fakeICAHandler) SubmitICATx(_, _ string, _ []byte) error                         { return nil }

// ----- Fee middleware tests -----

func TestFeePayload_Validate(t *testing.T) {
	good := FeePayload{RecvFee: "100", AckFee: "50", TimeoutFee: "0", Payee: "qor1xxx"}
	if err := good.Validate(); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*FeePayload)
	}{
		{"empty_recv", func(p *FeePayload) { p.RecvFee = "" }},
		{"empty_ack", func(p *FeePayload) { p.AckFee = "" }},
		{"empty_timeout", func(p *FeePayload) { p.TimeoutFee = "" }},
		{"empty_payee", func(p *FeePayload) { p.Payee = "" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := good
			c.mutate(&p)
			if err := p.Validate(); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestFeeMiddlewareHook_InterfaceShape(t *testing.T) {
	var _ FeeMiddlewareHook = (*fakeFeeHook)(nil)
}

type fakeFeeHook struct{}

func (fakeFeeHook) EscrowPacketFees(_ string, _ FeePayload) error  { return nil }
func (fakeFeeHook) PayPacketFee(_, _, _ string) error              { return nil }
func (fakeFeeHook) RefundUnusedFees(_ string) error                { return nil }

// ----- NFT tests -----

func TestNFTPacketData_Validate(t *testing.T) {
	good := NFTPacketData{
		ClassID:  "ibc/cosmoshub/transfer/cosmos1...",
		TokenIDs: []string{"token1", "token2"},
		Sender:   "cosmos1...",
		Receiver: "qor1...",
	}
	if err := good.Validate(); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
}

func TestNFTPacketData_Validate_RejectsEmptyClassID(t *testing.T) {
	p := NFTPacketData{TokenIDs: []string{"t1"}, Sender: "a", Receiver: "b"}
	if err := p.Validate(); err == nil {
		t.Error("expected error for empty class_id")
	}
}

func TestNFTPacketData_Validate_RejectsEmptyTokens(t *testing.T) {
	p := NFTPacketData{ClassID: "c", Sender: "a", Receiver: "b"}
	if err := p.Validate(); err == nil {
		t.Error("expected error for empty token_ids")
	}
}

func TestNFTPacketData_Validate_RejectsTokenURILengthMismatch(t *testing.T) {
	p := NFTPacketData{
		ClassID:   "c",
		TokenIDs:  []string{"t1", "t2"},
		TokenURIs: []string{"u1"}, // mismatch with TokenIDs
		Sender:    "a",
		Receiver:  "b",
	}
	if err := p.Validate(); err == nil {
		t.Error("expected error for token_uris length mismatch")
	}
}

func TestNFTPacketData_Validate_AllowsEmptyTokenURIs(t *testing.T) {
	p := NFTPacketData{
		ClassID:  "c",
		TokenIDs: []string{"t1", "t2"},
		// TokenURIs nil — represents "no per-token metadata", which is valid
		Sender:   "a",
		Receiver: "b",
	}
	if err := p.Validate(); err != nil {
		t.Errorf("expected ok with empty token_uris, got %v", err)
	}
}

func TestNFTHandlerHook_InterfaceShape(t *testing.T) {
	var _ NFTHandlerHook = (*fakeNFTHandler)(nil)
}

type fakeNFTHandler struct{}

func (fakeNFTHandler) SendNFT(_, _ string, _ NFTPacketData) error  { return nil }
func (fakeNFTHandler) OnRecvNFTPacket(_ NFTPacketData) error       { return nil }
