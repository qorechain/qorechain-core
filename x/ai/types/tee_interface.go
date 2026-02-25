package types

// TEE (Trusted Execution Environment) attestation interfaces for the AI module.
// These define the contract for future TEE-backed AI model execution, where
// model inference runs inside secure enclaves (SGX, TDX, SEV-SNP, ARM CCA)
// with verifiable attestation proofs on-chain.
//
// v1.1.0: Interface specification only — no implementation.

import (
	"time"
)

// TEEPlatform identifies the hardware TEE platform.
type TEEPlatform string

const (
	TEEPlatformSGX    TEEPlatform = "sgx"      // Intel SGX
	TEEPlatformTDX    TEEPlatform = "tdx"      // Intel TDX
	TEEPlatformSEVSNP TEEPlatform = "sev-snp"  // AMD SEV-SNP
	TEEPlatformCCA    TEEPlatform = "arm-cca"   // ARM CCA
)

// TEEEnclaveStatus represents the operational state of a TEE enclave.
type TEEEnclaveStatus struct {
	EnclaveID   string      `json:"enclave_id"`
	Platform    TEEPlatform `json:"platform"`
	Active      bool        `json:"active"`
	ModelLoaded bool        `json:"model_loaded"`
	ModelHash   []byte      `json:"model_hash,omitempty"`   // SHA-256 of loaded model weights
	MemoryUsage uint64      `json:"memory_usage"`           // bytes
	Uptime      int64       `json:"uptime"`                 // seconds
}

// TEEAttestation is the on-chain representation of a TEE attestation report.
// Validators verify this before accepting AI inference results.
type TEEAttestation struct {
	EnclaveID       string      `json:"enclave_id"`
	Platform        TEEPlatform `json:"platform"`
	AttestationData []byte      `json:"attestation_data"`     // Platform-specific attestation blob
	MeasurementHash []byte      `json:"measurement_hash"`     // MRENCLAVE / launch digest
	SignerHash      []byte      `json:"signer_hash"`          // MRSIGNER / author identity
	Timestamp       time.Time   `json:"timestamp"`
	Signature       []byte      `json:"signature"`            // Platform-signed attestation
	ReportData      []byte      `json:"report_data,omitempty"` // Custom data bound to attestation
}

// TEEExecutionResult captures the output of an AI model inference inside a TEE.
type TEEExecutionResult struct {
	EnclaveID     string         `json:"enclave_id"`
	ModelHash     []byte         `json:"model_hash"`
	InputHash     []byte         `json:"input_hash"`       // Hash of inference input
	Output        []byte         `json:"output"`           // Serialized inference output
	OutputHash    []byte         `json:"output_hash"`      // Hash of output for on-chain verification
	Attestation   TEEAttestation `json:"attestation"`      // Proof of execution
	GasUsed       uint64         `json:"gas_used"`
	ExecutionTime int64          `json:"execution_time_ms"`
}

// TEEVerifier validates TEE attestation reports. Each platform has its own
// verification logic (e.g., Intel attestation service for SGX, AMD key
// derivation for SEV-SNP).
type TEEVerifier interface {
	// VerifyAttestation checks a TEE attestation report is valid and recent.
	// Returns nil if the attestation is valid, or an error describing the failure.
	VerifyAttestation(attestation TEEAttestation) error

	// GetSupportedPlatforms returns the TEE platforms this verifier can handle.
	GetSupportedPlatforms() []TEEPlatform

	// IsEnclaveValid checks whether a specific enclave is in good standing
	// (not revoked, measurements match expected values).
	IsEnclaveValid(enclaveID string, measurementHash []byte) (bool, error)
}

// TEEExecutor runs AI inference inside a TEE enclave. Used by the AI sidecar
// to provide verifiable computation.
type TEEExecutor interface {
	// ExecuteInEnclave runs model inference on the given input inside a TEE.
	// Returns the result with attestation proof.
	ExecuteInEnclave(modelHash []byte, input []byte) (*TEEExecutionResult, error)

	// LoadModel loads AI model weights into the enclave's protected memory.
	// The model is identified by its content hash.
	LoadModel(modelHash []byte, modelData []byte) error

	// GetEnclaveStatus returns the current state of the TEE enclave.
	GetEnclaveStatus() (*TEEEnclaveStatus, error)

	// Attest generates a fresh attestation report for the current enclave state.
	// The reportData is bound into the attestation (e.g., a nonce or commitment).
	Attest(reportData []byte) (*TEEAttestation, error)
}
