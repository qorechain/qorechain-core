//go:build proprietary

package keeper

import (
	"testing"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

func TestGetPresetProfileDeFi(t *testing.T) {
	cfg := GetPresetProfile(types.ProfileDeFi)
	if cfg.Profile != types.ProfileDeFi {
		t.Errorf("expected profile %q, got %q", types.ProfileDeFi, cfg.Profile)
	}
	if cfg.SettlementMode != types.SettlementZK {
		t.Errorf("expected settlement %q, got %q", types.SettlementZK, cfg.SettlementMode)
	}
	if cfg.ProofConfig.System != types.ProofSystemSNARK {
		t.Errorf("expected proof system %q, got %q", types.ProofSystemSNARK, cfg.ProofConfig.System)
	}
	if cfg.BlockTimeMs != 500 {
		t.Errorf("expected block time 500ms, got %d", cfg.BlockTimeMs)
	}
	if cfg.VMType != "evm" {
		t.Errorf("expected VM type \"evm\", got %q", cfg.VMType)
	}
	if cfg.GasConfig.GasModel != "eip1559" {
		t.Errorf("expected gas model \"eip1559\", got %q", cfg.GasConfig.GasModel)
	}
	if cfg.DABackend != types.DANative {
		t.Errorf("expected DA backend %q, got %q", types.DANative, cfg.DABackend)
	}
	if cfg.SequencerConfig.Mode != types.SequencerDedicated {
		t.Errorf("expected sequencer %q, got %q", types.SequencerDedicated, cfg.SequencerConfig.Mode)
	}
}

func TestGetPresetProfileGaming(t *testing.T) {
	cfg := GetPresetProfile(types.ProfileGaming)
	if cfg.Profile != types.ProfileGaming {
		t.Errorf("expected profile %q, got %q", types.ProfileGaming, cfg.Profile)
	}
	if cfg.SettlementMode != types.SettlementBased {
		t.Errorf("expected settlement %q, got %q", types.SettlementBased, cfg.SettlementMode)
	}
	if cfg.BlockTimeMs != 200 {
		t.Errorf("expected block time 200ms, got %d", cfg.BlockTimeMs)
	}
	if cfg.SequencerConfig.Mode != types.SequencerBased {
		t.Errorf("expected sequencer %q, got %q", types.SequencerBased, cfg.SequencerConfig.Mode)
	}
	if cfg.VMType != "custom" {
		t.Errorf("expected VM type \"custom\", got %q", cfg.VMType)
	}
	if cfg.GasConfig.GasModel != "flat" {
		t.Errorf("expected gas model \"flat\", got %q", cfg.GasConfig.GasModel)
	}
	if cfg.MaxTxPerBlock != 50000 {
		t.Errorf("expected max tx 50000, got %d", cfg.MaxTxPerBlock)
	}
}

func TestGetPresetProfileNFT(t *testing.T) {
	cfg := GetPresetProfile(types.ProfileNFT)
	if cfg.Profile != types.ProfileNFT {
		t.Errorf("expected profile %q, got %q", types.ProfileNFT, cfg.Profile)
	}
	if cfg.SettlementMode != types.SettlementOptimistic {
		t.Errorf("expected settlement %q, got %q", types.SettlementOptimistic, cfg.SettlementMode)
	}
	if cfg.ProofConfig.System != types.ProofSystemFraud {
		t.Errorf("expected proof system %q, got %q", types.ProofSystemFraud, cfg.ProofConfig.System)
	}
	if cfg.DABackend != types.DACelestia {
		t.Errorf("expected DA backend %q, got %q", types.DACelestia, cfg.DABackend)
	}
	if cfg.VMType != "cosmwasm" {
		t.Errorf("expected VM type \"cosmwasm\", got %q", cfg.VMType)
	}
	if cfg.BlockTimeMs != 2000 {
		t.Errorf("expected block time 2000ms, got %d", cfg.BlockTimeMs)
	}
	if cfg.ProofConfig.ChallengeWindowSec != 604800 {
		t.Errorf("expected challenge window 604800s, got %d", cfg.ProofConfig.ChallengeWindowSec)
	}
}

func TestGetPresetProfileEnterprise(t *testing.T) {
	cfg := GetPresetProfile(types.ProfileEnterprise)
	if cfg.Profile != types.ProfileEnterprise {
		t.Errorf("expected profile %q, got %q", types.ProfileEnterprise, cfg.Profile)
	}
	if cfg.SettlementMode != types.SettlementBased {
		t.Errorf("expected settlement %q, got %q", types.SettlementBased, cfg.SettlementMode)
	}
	if cfg.SequencerConfig.Mode != types.SequencerBased {
		t.Errorf("expected sequencer %q, got %q", types.SequencerBased, cfg.SequencerConfig.Mode)
	}
	if cfg.GasConfig.GasModel != "subsidized" {
		t.Errorf("expected gas model \"subsidized\", got %q", cfg.GasConfig.GasModel)
	}
	if cfg.GasConfig.BaseGasPrice != "0.0" {
		t.Errorf("expected base gas price \"0.0\", got %q", cfg.GasConfig.BaseGasPrice)
	}
	if cfg.VMType != "evm" {
		t.Errorf("expected VM type \"evm\", got %q", cfg.VMType)
	}
	if cfg.BlockTimeMs != 1000 {
		t.Errorf("expected block time 1000ms, got %d", cfg.BlockTimeMs)
	}
}

func TestGetPresetProfileCustomFallback(t *testing.T) {
	cfg := GetPresetProfile(types.ProfileCustom)
	if cfg.Profile != types.ProfileCustom {
		t.Errorf("expected profile %q, got %q", types.ProfileCustom, cfg.Profile)
	}
	if cfg.SettlementMode != types.SettlementOptimistic {
		t.Errorf("expected settlement %q, got %q", types.SettlementOptimistic, cfg.SettlementMode)
	}
	if cfg.BlockTimeMs != 1000 {
		t.Errorf("expected block time 1000ms, got %d", cfg.BlockTimeMs)
	}
}
