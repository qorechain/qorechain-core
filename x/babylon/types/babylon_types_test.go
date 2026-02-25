package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBTCRestakingConfigDefault(t *testing.T) {
	cfg := DefaultBTCRestakingConfig()

	if cfg.Enabled {
		t.Error("expected Enabled to be false by default")
	}
	if cfg.MinStakeAmount <= 0 {
		t.Error("expected MinStakeAmount to be positive")
	}
	if cfg.UnbondingPeriod <= 0 {
		t.Error("expected UnbondingPeriod to be positive")
	}
	if cfg.CheckpointInterval <= 0 {
		t.Error("expected CheckpointInterval to be positive")
	}
	if cfg.BabylonChainID == "" {
		t.Error("expected BabylonChainID to be non-empty")
	}
}

func TestGenesisStateValidation(t *testing.T) {
	gs := DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Fatalf("expected DefaultGenesisState().Validate() to return nil, got: %v", err)
	}
}

func TestBTCStakingPositionFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	pos := BTCStakingPosition{
		ID:              "pos-001",
		StakerAddress:   "qor1abc123",
		BTCTxHash:       "0xdeadbeef",
		AmountSatoshis:  500000,
		StakedAt:        now,
		UnbondingHeight: 1000,
		Status:          "active",
		ValidatorAddr:   "qorvaloper1xyz",
	}

	data, err := json.Marshal(pos)
	if err != nil {
		t.Fatalf("failed to marshal BTCStakingPosition: %v", err)
	}

	var decoded BTCStakingPosition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal BTCStakingPosition: %v", err)
	}

	if decoded.ID != pos.ID {
		t.Errorf("ID mismatch: expected %q, got %q", pos.ID, decoded.ID)
	}
	if decoded.StakerAddress != pos.StakerAddress {
		t.Errorf("StakerAddress mismatch: expected %q, got %q", pos.StakerAddress, decoded.StakerAddress)
	}
	if decoded.BTCTxHash != pos.BTCTxHash {
		t.Errorf("BTCTxHash mismatch: expected %q, got %q", pos.BTCTxHash, decoded.BTCTxHash)
	}
	if decoded.AmountSatoshis != pos.AmountSatoshis {
		t.Errorf("AmountSatoshis mismatch: expected %d, got %d", pos.AmountSatoshis, decoded.AmountSatoshis)
	}
	if !decoded.StakedAt.Equal(pos.StakedAt) {
		t.Errorf("StakedAt mismatch: expected %v, got %v", pos.StakedAt, decoded.StakedAt)
	}
	if decoded.UnbondingHeight != pos.UnbondingHeight {
		t.Errorf("UnbondingHeight mismatch: expected %d, got %d", pos.UnbondingHeight, decoded.UnbondingHeight)
	}
	if decoded.Status != pos.Status {
		t.Errorf("Status mismatch: expected %q, got %q", pos.Status, decoded.Status)
	}
	if decoded.ValidatorAddr != pos.ValidatorAddr {
		t.Errorf("ValidatorAddr mismatch: expected %q, got %q", pos.ValidatorAddr, decoded.ValidatorAddr)
	}
}
