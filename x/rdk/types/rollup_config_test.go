package types

import (
	"encoding/json"
	"testing"
)

func TestDefaultSequencerConfig(t *testing.T) {
	cfg := DefaultSequencerConfig()
	if cfg.Mode != SequencerDedicated {
		t.Errorf("expected mode %q, got %q", SequencerDedicated, cfg.Mode)
	}
	if cfg.SharedSetMinSize != 1 {
		t.Errorf("expected SharedSetMinSize 1, got %d", cfg.SharedSetMinSize)
	}
	if cfg.InclusionDelay != 10 {
		t.Errorf("expected InclusionDelay 10, got %d", cfg.InclusionDelay)
	}
	if cfg.PriorityFeeShare != "0.0" {
		t.Errorf("expected PriorityFeeShare \"0.0\", got %q", cfg.PriorityFeeShare)
	}
}

func TestDefaultProofConfig(t *testing.T) {
	cfg := DefaultProofConfig()
	if cfg.System != ProofSystemFraud {
		t.Errorf("expected system %q, got %q", ProofSystemFraud, cfg.System)
	}
	if cfg.ChallengeWindowSec != 604800 {
		t.Errorf("expected ChallengeWindowSec 604800, got %d", cfg.ChallengeWindowSec)
	}
	if cfg.ChallengeBond != 1000000000 {
		t.Errorf("expected ChallengeBond 1000000000, got %d", cfg.ChallengeBond)
	}
	if cfg.MaxProofSize != 1048576 {
		t.Errorf("expected MaxProofSize 1048576, got %d", cfg.MaxProofSize)
	}
	if cfg.RecursionDepth != 1 {
		t.Errorf("expected RecursionDepth 1, got %d", cfg.RecursionDepth)
	}
}

func TestDefaultRollupGasConfig(t *testing.T) {
	cfg := DefaultRollupGasConfig()
	if cfg.GasModel != "standard" {
		t.Errorf("expected GasModel \"standard\", got %q", cfg.GasModel)
	}
	if cfg.BaseGasPrice != "0.001" {
		t.Errorf("expected BaseGasPrice \"0.001\", got %q", cfg.BaseGasPrice)
	}
	if cfg.MaxGasLimit != 10000000 {
		t.Errorf("expected MaxGasLimit 10000000, got %d", cfg.MaxGasLimit)
	}
}

func TestRollupProfileValues(t *testing.T) {
	profiles := []RollupProfile{ProfileDeFi, ProfileGaming, ProfileNFT, ProfileEnterprise, ProfileCustom}
	expected := []string{"defi", "gaming", "nft", "enterprise", "custom"}

	if len(profiles) != 5 {
		t.Fatalf("expected 5 profiles, got %d", len(profiles))
	}
	for i, p := range profiles {
		if string(p) != expected[i] {
			t.Errorf("profile %d: expected %q, got %q", i, expected[i], string(p))
		}
	}
}

func TestSettlementModeValues(t *testing.T) {
	modes := []SettlementMode{SettlementOptimistic, SettlementZK, SettlementBased, SettlementSovereign}
	expected := []string{"optimistic", "zk", "based", "sovereign"}

	for i, m := range modes {
		if string(m) != expected[i] {
			t.Errorf("mode %d: expected %q, got %q", i, expected[i], string(m))
		}
	}
}

func TestSequencerModeValues(t *testing.T) {
	modes := []SequencerMode{SequencerDedicated, SequencerShared, SequencerBased}
	expected := []string{"dedicated", "shared", "based"}

	for i, m := range modes {
		if string(m) != expected[i] {
			t.Errorf("mode %d: expected %q, got %q", i, expected[i], string(m))
		}
	}
}

func TestProofSystemValues(t *testing.T) {
	systems := []ProofSystem{ProofSystemFraud, ProofSystemSNARK, ProofSystemSTARK, ProofSystemNone}
	expected := []string{"fraud", "snark", "stark", "none"}

	for i, s := range systems {
		if string(s) != expected[i] {
			t.Errorf("system %d: expected %q, got %q", i, expected[i], string(s))
		}
	}
}

func TestDABackendValues(t *testing.T) {
	backends := []DABackend{DANative, DACelestia, DABoth}
	expected := []string{"native", "celestia", "both"}

	for i, b := range backends {
		if string(b) != expected[i] {
			t.Errorf("backend %d: expected %q, got %q", i, expected[i], string(b))
		}
	}
}

func TestRollupConfigValidation(t *testing.T) {
	// Valid optimistic config
	valid := RollupConfig{
		RollupID:       "test-rollup",
		Creator:        "qor1abc",
		Profile:        ProfileDeFi,
		SettlementMode: SettlementOptimistic,
		SequencerConfig: SequencerConfig{
			Mode: SequencerDedicated,
		},
		DABackend:     DANative,
		BlockTimeMs:   1000,
		MaxTxPerBlock: 10000,
		GasConfig:     DefaultRollupGasConfig(),
		VMType:        "evm",
		ProofConfig: ProofConfig{
			System:             ProofSystemFraud,
			ChallengeWindowSec: 604800,
		},
		StakeAmount: 10000000000,
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("expected valid config to pass, got: %v", err)
	}

	// Zero block time
	bad := valid
	bad.BlockTimeMs = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero block time")
	}

	// Zero max tx
	bad = valid
	bad.MaxTxPerBlock = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero max tx per block")
	}

	// Zero stake
	bad = valid
	bad.StakeAmount = 0
	if err := bad.Validate(); err == nil {
		t.Error("expected error for zero stake amount")
	}

	// Negative stake
	bad = valid
	bad.StakeAmount = -1
	if err := bad.Validate(); err == nil {
		t.Error("expected error for negative stake amount")
	}
}

func TestSettlementSequencerProofCompatibility(t *testing.T) {
	base := RollupConfig{
		RollupID:      "compat-test",
		Creator:       "qor1abc",
		Profile:       ProfileCustom,
		DABackend:     DANative,
		BlockTimeMs:   1000,
		MaxTxPerBlock: 10000,
		GasConfig:     DefaultRollupGasConfig(),
		VMType:        "evm",
		StakeAmount:   10000000000,
	}

	tests := []struct {
		name       string
		settlement SettlementMode
		sequencer  SequencerMode
		proof      ProofSystem
		wantErr    bool
	}{
		{"based+based=ok", SettlementBased, SequencerBased, ProofSystemNone, false},
		{"based+dedicated=fail", SettlementBased, SequencerDedicated, ProofSystemNone, true},
		{"zk+snark=ok", SettlementZK, SequencerDedicated, ProofSystemSNARK, false},
		{"zk+stark=ok", SettlementZK, SequencerDedicated, ProofSystemSTARK, false},
		{"zk+fraud=fail", SettlementZK, SequencerDedicated, ProofSystemFraud, true},
		{"optimistic+fraud=ok", SettlementOptimistic, SequencerDedicated, ProofSystemFraud, false},
		{"optimistic+snark=fail", SettlementOptimistic, SequencerDedicated, ProofSystemSNARK, true},
		{"sovereign+none=ok", SettlementSovereign, SequencerDedicated, ProofSystemNone, false},
		{"sovereign+fraud=fail", SettlementSovereign, SequencerDedicated, ProofSystemFraud, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			cfg.SettlementMode = tc.settlement
			cfg.SequencerConfig.Mode = tc.sequencer
			cfg.ProofConfig.System = tc.proof
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error for %s, got: %v", tc.name, err)
			}
		})
	}
}

func TestRollupConfigJSONRoundtrip(t *testing.T) {
	cfg := RollupConfig{
		RollupID:        "json-test",
		Creator:         "qor1xyz",
		Profile:         ProfileDeFi,
		SettlementMode:  SettlementZK,
		SequencerConfig: DefaultSequencerConfig(),
		DABackend:       DANative,
		BlockTimeMs:     500,
		MaxTxPerBlock:   10000,
		GasConfig:       DefaultRollupGasConfig(),
		VMType:          "evm",
		ProofConfig:     DefaultProofConfig(),
		Status:          RollupStatusActive,
		StakeAmount:     10000000000,
		LayerID:         "layer-1",
		CreatedHeight:   100,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RollupConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.RollupID != cfg.RollupID {
		t.Errorf("RollupID mismatch: expected %q, got %q", cfg.RollupID, decoded.RollupID)
	}
	if decoded.SettlementMode != cfg.SettlementMode {
		t.Errorf("SettlementMode mismatch: expected %q, got %q", cfg.SettlementMode, decoded.SettlementMode)
	}
	if decoded.BlockTimeMs != cfg.BlockTimeMs {
		t.Errorf("BlockTimeMs mismatch: expected %d, got %d", cfg.BlockTimeMs, decoded.BlockTimeMs)
	}
	if decoded.Status != cfg.Status {
		t.Errorf("Status mismatch: expected %q, got %q", cfg.Status, decoded.Status)
	}
}
