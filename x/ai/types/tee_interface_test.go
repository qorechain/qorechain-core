package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTEEPlatformConstants(t *testing.T) {
	platforms := []TEEPlatform{TEEPlatformSGX, TEEPlatformTDX, TEEPlatformSEVSNP, TEEPlatformCCA}
	seen := make(map[TEEPlatform]bool)
	for _, p := range platforms {
		if p == "" {
			t.Error("platform constant should not be empty")
		}
		if seen[p] {
			t.Errorf("duplicate platform constant: %s", p)
		}
		seen[p] = true
	}
}

func TestTEEAttestation_JSONMarshal(t *testing.T) {
	att := TEEAttestation{
		EnclaveID:       "enclave-001",
		Platform:        TEEPlatformSGX,
		AttestationData: []byte{0x01, 0x02, 0x03},
		MeasurementHash: make([]byte, 32),
		SignerHash:       make([]byte, 32),
		Timestamp:       time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC),
		Signature:       []byte{0xAA, 0xBB},
		ReportData:      []byte{0xCC},
	}

	bz, err := json.Marshal(att)
	if err != nil {
		t.Fatalf("failed to marshal TEEAttestation: %v", err)
	}

	var decoded TEEAttestation
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("failed to unmarshal TEEAttestation: %v", err)
	}

	if decoded.EnclaveID != att.EnclaveID {
		t.Errorf("EnclaveID mismatch: %s != %s", decoded.EnclaveID, att.EnclaveID)
	}
	if decoded.Platform != att.Platform {
		t.Errorf("Platform mismatch: %s != %s", decoded.Platform, att.Platform)
	}
}

func TestTEEEnclaveStatus_JSONMarshal(t *testing.T) {
	status := TEEEnclaveStatus{
		EnclaveID:   "enc-002",
		Platform:    TEEPlatformTDX,
		Active:      true,
		ModelLoaded: true,
		ModelHash:   make([]byte, 32),
		MemoryUsage: 1024 * 1024 * 512,
		Uptime:      3600,
	}

	bz, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal TEEEnclaveStatus: %v", err)
	}

	var decoded TEEEnclaveStatus
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("failed to unmarshal TEEEnclaveStatus: %v", err)
	}

	if !decoded.Active {
		t.Error("Active should be true after roundtrip")
	}
	if decoded.MemoryUsage != status.MemoryUsage {
		t.Errorf("MemoryUsage mismatch: %d != %d", decoded.MemoryUsage, status.MemoryUsage)
	}
}

func TestTEEExecutionResult_JSONMarshal(t *testing.T) {
	result := TEEExecutionResult{
		EnclaveID:     "enc-003",
		ModelHash:     make([]byte, 32),
		InputHash:     make([]byte, 32),
		Output:        []byte("inference output"),
		OutputHash:    make([]byte, 32),
		GasUsed:       50000,
		ExecutionTime: 150,
		Attestation: TEEAttestation{
			EnclaveID: "enc-003",
			Platform:  TEEPlatformSEVSNP,
			Timestamp: time.Now().UTC(),
		},
	}

	bz, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal TEEExecutionResult: %v", err)
	}

	var decoded TEEExecutionResult
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("failed to unmarshal TEEExecutionResult: %v", err)
	}

	if decoded.GasUsed != result.GasUsed {
		t.Errorf("GasUsed mismatch: %d != %d", decoded.GasUsed, result.GasUsed)
	}
	if decoded.Attestation.Platform != TEEPlatformSEVSNP {
		t.Errorf("nested attestation platform mismatch: %s", decoded.Attestation.Platform)
	}
}
