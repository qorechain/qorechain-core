package types

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestParseStoredAmountAcceptsZeroRefusesNegative(t *testing.T) {
	for in, ok := range map[string]bool{
		"0": true, "1": true, "10000000000000": true,
		"-1": false, "": false, "abc": false, "1.5": false,
	} {
		_, err := ParseStoredAmount(in)
		if (err == nil) != ok {
			t.Errorf("ParseStoredAmount(%q): err=%v, want ok=%v", in, err, ok)
		}
	}
	// The message parser keeps refusing zero: that is the boundary it guards.
	if _, err := ParseAmount("0"); err == nil {
		t.Fatal("ParseAmount accepted zero")
	}
}

func TestBridgeStatusExactSpelling(t *testing.T) {
	for s, ok := range map[BridgeStatus]bool{
		"active": true, "paused": true, "pending": true,
		"PAUSED": false, "Paused": false, "paused ": false, "": false, "open": false,
	} {
		if s.IsValid() != ok {
			t.Errorf("%q.IsValid() = %v, want %v", s, !ok, ok)
		}
	}
}

func TestIdentifiersRefuseTheSeparator(t *testing.T) {
	for _, bad := range []string{"eth/x", "a/b", "", strings.Repeat("a", 65), "ETHEREUM", "chain id"} {
		if ValidateChainID(bad) == nil {
			t.Errorf("chain id %q accepted", bad)
		}
	}
	for _, good := range []string{"ethereum", "zksync_era", "bsc", "a-b.c"} {
		if err := ValidateChainID(good); err != nil {
			t.Errorf("chain id %q refused: %v", good, err)
		}
	}
	for _, bad := range []string{"USDC/ETH", "", "US DC"} {
		if ValidateAsset(bad) == nil {
			t.Errorf("asset %q accepted", bad)
		}
	}
	for _, good := range []string{"USDC", "ETH", "S", "wBTC"} {
		if err := ValidateAsset(good); err != nil {
			t.Errorf("asset %q refused: %v", good, err)
		}
	}
}

func TestLimitsAndLedgerInvariants(t *testing.T) {
	cb := CircuitBreakerState{Chain: "ethereum", MaxSingleTransfer: "0", DailyLimit: "0", CurrentDaily: "0"}
	if err := cb.Validate(); err != nil {
		t.Fatalf("unlimited breaker refused: %v", err)
	}
	cb.MaxSingleTransfer, cb.DailyLimit = "11", "10"
	if cb.Validate() == nil {
		t.Fatal("max single above daily accepted")
	}
	cb.MaxSingleTransfer, cb.DailyLimit = "11", "0" // daily unlimited
	if err := cb.Validate(); err != nil {
		t.Fatalf("single limit with unlimited daily refused: %v", err)
	}
	cb.CurrentDaily = "-1"
	if cb.Validate() == nil {
		t.Fatal("negative daily counter accepted")
	}

	la := LockedAmount{Chain: "ethereum", Asset: "USDC", TotalLocked: "0", TotalMinted: "0"}
	if err := la.Validate(); err != nil {
		t.Fatalf("empty ledger refused: %v", err)
	}
	la.TotalMinted = "1"
	if la.Validate() == nil {
		t.Fatal("minted above locked accepted")
	}
}

func key(n int, b byte) []byte { return bytes.Repeat([]byte{b}, n) }

func TestVerifierSets(t *testing.T) {
	good := [][]byte{key(32, 1), key(32, 2), key(32, 3)}
	if err := ValidateEd25519Set(good, 2); err != nil {
		t.Fatalf("good ed25519 set refused: %v", err)
	}
	for name, c := range map[string]struct {
		keys [][]byte
		t    int
	}{
		"31-byte key":   {[][]byte{key(32, 1), key(31, 2)}, 1},
		"33-byte key":   {[][]byte{key(33, 1)}, 1},
		"duplicate":     {[][]byte{key(32, 1), key(32, 1)}, 1},
		"threshold 0":   {good, 0},
		"threshold n+1": {good, 4},
		"empty":         {nil, 1},
	} {
		if ValidateEd25519Set(c.keys, c.t) == nil {
			t.Errorf("ed25519 %s accepted", name)
		}
	}
	if ValidateBLSCommitteeShape([][]byte{key(48, 1)}, 1) == nil {
		t.Error("48-byte BLS key accepted for a G2 committee")
	}

	guardians := make([][]byte, 19)
	for i := range guardians {
		guardians[i] = key(20, byte(i+1))
	}
	if err := ValidateWormholeGuardianSet(guardians, 13); err != nil {
		t.Fatalf("19 guardians with quorum 13 refused: %v", err)
	}
	if ValidateWormholeGuardianSet(guardians, 12) == nil {
		t.Error("quorum below the Wormhole rule accepted")
	}
	if ValidateWormholeGuardianSet([][]byte{key(32, 1)}, 1) == nil {
		t.Error("32-byte guardian address accepted")
	}
}

// The historical genesis files of both networks must still validate: a node
// that runs `genesis validate` on them, or a tool that does, must not be told
// its network's genesis is invalid.
func TestRealGenesisFilesStillValidate(t *testing.T) {
	for _, f := range []string{"testdata/mainnet_bridge_genesis.json", "testdata/diana_bridge_genesis.json"} {
		bz, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var gs GenesisState
		if err := json.Unmarshal(bz, &gs); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if len(gs.ChainConfigs) != 37 {
			t.Fatalf("%s: %d chain configs, want 37", f, len(gs.ChainConfigs))
		}
		if err := gs.Validate(); err != nil {
			t.Fatalf("%s no longer validates: %v", f, err)
		}
	}
	if err := DefaultGenesisState().Validate(); err != nil {
		t.Fatalf("default genesis no longer validates: %v", err)
	}
}

func TestCraftedGenesisRefused(t *testing.T) {
	for name, mutate := range map[string]func(*GenesisState){
		"duplicate chain id":    func(g *GenesisState) { g.ChainConfigs = append(g.ChainConfigs, g.ChainConfigs[0]) },
		"status casing":         func(g *GenesisState) { g.ChainConfigs[0].Status = "PAUSED" },
		"separator in chain id": func(g *GenesisState) { g.ChainConfigs[0].ChainID = "eth/x" },
		"separator in asset":    func(g *GenesisState) { g.ChainConfigs[0].SupportedAssets = []string{"A/B"} },
		"single above daily": func(g *GenesisState) {
			g.ChainConfigs[0].MaxSingleTransfer, g.ChainConfigs[0].DailyLimit = "10", "1"
		},
		"minted above locked": func(g *GenesisState) {
			g.Locked = []LockedAmount{{Chain: "ethereum", Asset: "USDC", TotalLocked: "1", TotalMinted: "2"}}
		},
		"negative breaker": func(g *GenesisState) {
			g.Breakers = []CircuitBreakerState{{Chain: "ethereum", MaxSingleTransfer: "-1", DailyLimit: "0", CurrentDaily: "0"}}
		},
		"unparseable threshold": func(g *GenesisState) { g.Config.LargeTransferThreshold = "lots" },
	} {
		gs := DefaultGenesisState()
		mutate(gs)
		if gs.Validate() == nil {
			t.Errorf("%s: genesis accepted", name)
		}
	}
}

func TestUpdateCircuitBreakerLimitsValidateBasic(t *testing.T) {
	admin := sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String()
	ok := MsgUpdateCircuitBreakerLimits{Admin: admin, ChainId: "ethereum", MaxSingleTransfer: "5", DailyLimit: "10"}
	if err := ok.ValidateBasic(); err != nil {
		t.Fatalf("valid message refused: %v", err)
	}
	for name, m := range map[string]MsgUpdateCircuitBreakerLimits{
		"bad admin":      {Admin: "x", ChainId: "ethereum", MaxSingleTransfer: "1", DailyLimit: "1"},
		"bad chain":      {Admin: admin, ChainId: "Eth/x", MaxSingleTransfer: "1", DailyLimit: "1"},
		"single > daily": {Admin: admin, ChainId: "ethereum", MaxSingleTransfer: "11", DailyLimit: "10"},
		"negative":       {Admin: admin, ChainId: "ethereum", MaxSingleTransfer: "-1", DailyLimit: "10"},
		"empty limit":    {Admin: admin, ChainId: "ethereum", MaxSingleTransfer: "", DailyLimit: "10"},
	} {
		if m.ValidateBasic() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
