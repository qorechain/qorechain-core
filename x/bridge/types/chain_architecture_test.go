package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIsValidChainArchitecture(t *testing.T) {
	cases := []struct {
		in   ChainArchitecture
		want bool
	}{
		{ChainArchEmpty, true},
		{ChainArchIBCClassic, true},
		{ChainArchIBCEurekaV2, true},
		{ChainArchitecture("ibc_v3"), false},
		{ChainArchitecture("classic"), false},   // missing prefix
		{ChainArchitecture("IBC_CLASSIC"), false}, // case-sensitive
	}
	for _, c := range cases {
		if got := IsValidChainArchitecture(c.in); got != c.want {
			t.Errorf("IsValidChainArchitecture(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestChainConfig_IBCFields_OmitemptyWireFormat — for non-IBC chains the
// new IBC-specific fields must NOT appear in the JSON wire format.
// This is critical: external integrations parse ChainConfig and would
// break if a stable chain config gained a flurry of empty IBC fields.
func TestChainConfig_IBCFields_OmitemptyWireFormat(t *testing.T) {
	ethereum := DefaultChainConfigs()[0] // first entry is Ethereum
	if ethereum.ChainType != ChainTypeEVM {
		t.Skip("expected ethereum first; test out of date")
	}

	bz, err := json.Marshal(ethereum)
	if err != nil {
		t.Fatal(err)
	}
	wireText := string(bz)

	// None of the new IBC fields should appear for an EVM chain.
	for _, field := range []string{
		"\"architecture\"",
		"\"ibc_channel_id\"",
		"\"ibc_port_id\"",
		"\"ibc_connection_id\"",
		"\"eureka_client_type\"",
	} {
		if strings.Contains(wireText, field) {
			t.Errorf("EVM ChainConfig wire format unexpectedly contains %s: %s", field, wireText)
		}
	}
}

// TestChainConfig_IBCFields_RoundTrip — when an IBC chain has the new
// fields populated, they survive a JSON round trip.
func TestChainConfig_IBCFields_RoundTrip(t *testing.T) {
	original := ChainConfig{
		ChainID:          "cosmoshub",
		Name:             "Cosmos Hub",
		ChainType:        ChainTypeIBC,
		Status:           BridgeStatusPending,
		MinConfirmations: 1,
		SupportedAssets:  []string{"ATOM"},
		MaxSingleTransfer: "1000000000000",
		DailyLimit:       "10000000000000",

		Architecture:     ChainArchIBCEurekaV2,
		IBCChannelID:     "channel-0",
		IBCPortID:        "transfer",
		IBCConnectionID:  "connection-0",
		EurekaClientType: "tendermint",
	}

	bz, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var round ChainConfig
	if err := json.Unmarshal(bz, &round); err != nil {
		t.Fatal(err)
	}
	if round.Architecture != original.Architecture {
		t.Errorf("Architecture lost: got %q, want %q", round.Architecture, original.Architecture)
	}
	if round.IBCChannelID != original.IBCChannelID {
		t.Errorf("IBCChannelID lost: got %q", round.IBCChannelID)
	}
	if round.EurekaClientType != original.EurekaClientType {
		t.Errorf("EurekaClientType lost: got %q", round.EurekaClientType)
	}
}

// TestChainConfig_NonIBCDefaultsLeaveArchEmpty — every default config
// of a non-IBC chain must have empty Architecture, since the field is
// IBC-specific.
func TestChainConfig_NonIBCDefaultsLeaveArchEmpty(t *testing.T) {
	for _, c := range DefaultChainConfigs() {
		if c.ChainType != ChainTypeIBC && c.Architecture != ChainArchEmpty {
			t.Errorf("non-IBC chain %q has non-empty Architecture %q", c.ChainID, c.Architecture)
		}
	}
}
