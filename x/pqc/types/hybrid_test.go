package types

import (
	"testing"
)

func TestHybridSignatureMode_Constants(t *testing.T) {
	if HybridDisabled != 0 {
		t.Errorf("HybridDisabled should be 0, got %d", HybridDisabled)
	}
	if HybridOptional != 1 {
		t.Errorf("HybridOptional should be 1, got %d", HybridOptional)
	}
	if HybridRequired != 2 {
		t.Errorf("HybridRequired should be 2, got %d", HybridRequired)
	}
}

func TestHybridSignatureMode_IsValid(t *testing.T) {
	tests := []struct {
		mode  HybridSignatureMode
		valid bool
	}{
		{HybridDisabled, true},
		{HybridOptional, true},
		{HybridRequired, true},
		{HybridSignatureMode(3), false},
		{HybridSignatureMode(100), false},
	}

	for _, tc := range tests {
		if tc.mode.IsValid() != tc.valid {
			t.Errorf("mode %d: expected IsValid()=%v, got %v", tc.mode, tc.valid, tc.mode.IsValid())
		}
	}
}

func TestHybridSignatureMode_String(t *testing.T) {
	tests := []struct {
		mode     HybridSignatureMode
		expected string
	}{
		{HybridDisabled, "disabled"},
		{HybridOptional, "optional"},
		{HybridRequired, "required"},
		{HybridSignatureMode(99), "unknown_99"},
	}

	for _, tc := range tests {
		if tc.mode.String() != tc.expected {
			t.Errorf("mode %d: expected String()=%q, got %q", tc.mode, tc.expected, tc.mode.String())
		}
	}
}

func TestHybridSignatureMode_Description(t *testing.T) {
	// Just verify descriptions are non-empty for valid modes
	for _, mode := range []HybridSignatureMode{HybridDisabled, HybridOptional, HybridRequired} {
		desc := mode.Description()
		if desc == "" {
			t.Errorf("mode %d: Description() returned empty string", mode)
		}
	}
}

func TestHybridSignatureModeFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected HybridSignatureMode
		hasErr   bool
	}{
		{"disabled", HybridDisabled, false},
		{"optional", HybridOptional, false},
		{"required", HybridRequired, false},
		{"DISABLED", HybridDisabled, false},
		{"OPTIONAL", HybridOptional, false},
		{"REQUIRED", HybridRequired, false},
		{"0", HybridDisabled, false},
		{"1", HybridOptional, false},
		{"2", HybridRequired, false},
		{"Optional", 0, true},   // mixed case not supported
		{"invalid", 0, true},
		{"", 0, true},
		{"3", 0, true},
	}

	for _, tc := range tests {
		mode, err := HybridSignatureModeFromString(tc.input)
		if tc.hasErr {
			if err == nil {
				t.Errorf("input %q: expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("input %q: unexpected error: %v", tc.input, err)
			}
			if mode != tc.expected {
				t.Errorf("input %q: expected %d, got %d", tc.input, tc.expected, mode)
			}
		}
	}
}

func TestPQCHybridSignature_Validate(t *testing.T) {
	tests := []struct {
		name   string
		sig    PQCHybridSignature
		hasErr bool
	}{
		{
			name: "valid Dilithium5 signature",
			sig: PQCHybridSignature{
				AlgorithmID:  AlgorithmDilithium5,
				PQCSignature: make([]byte, 4627),
			},
			hasErr: false,
		},
		{
			name: "valid with optional pubkey",
			sig: PQCHybridSignature{
				AlgorithmID:  AlgorithmDilithium5,
				PQCSignature: make([]byte, 4627),
				PQCPublicKey: make([]byte, 2592),
			},
			hasErr: false,
		},
		{
			name: "empty signature",
			sig: PQCHybridSignature{
				AlgorithmID:  AlgorithmDilithium5,
				PQCSignature: nil,
			},
			hasErr: true,
		},
		{
			name: "KEM algorithm rejected",
			sig: PQCHybridSignature{
				AlgorithmID:  AlgorithmMLKEM1024,
				PQCSignature: make([]byte, 100),
			},
			hasErr: true,
		},
		{
			name: "wrong Dilithium5 sig size",
			sig: PQCHybridSignature{
				AlgorithmID:  AlgorithmDilithium5,
				PQCSignature: make([]byte, 100),
			},
			hasErr: true,
		},
		{
			name: "wrong Dilithium5 pubkey size",
			sig: PQCHybridSignature{
				AlgorithmID:  AlgorithmDilithium5,
				PQCSignature: make([]byte, 4627),
				PQCPublicKey: make([]byte, 100),
			},
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.sig.Validate()
			if tc.hasErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.hasErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestPQCHybridSignature_HasPublicKey(t *testing.T) {
	sig1 := PQCHybridSignature{PQCPublicKey: nil}
	if sig1.HasPublicKey() {
		t.Error("nil PQCPublicKey should return false")
	}

	sig2 := PQCHybridSignature{PQCPublicKey: []byte{}}
	if sig2.HasPublicKey() {
		t.Error("empty PQCPublicKey should return false")
	}

	sig3 := PQCHybridSignature{PQCPublicKey: []byte{0x01}}
	if !sig3.HasPublicKey() {
		t.Error("non-empty PQCPublicKey should return true")
	}
}

func TestPQCHybridSignature_String(t *testing.T) {
	sig := PQCHybridSignature{
		AlgorithmID:  AlgorithmDilithium5,
		PQCSignature: make([]byte, 4627),
	}
	s := sig.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}

func TestHybridSigTypeURL(t *testing.T) {
	if HybridSigTypeURL == "" {
		t.Error("HybridSigTypeURL should not be empty")
	}
	if HybridSigTypeURL != "/qorechain.pqc.v1.PQCHybridSignature" {
		t.Errorf("unexpected TypeURL: %s", HybridSigTypeURL)
	}
}

func TestDefaultParams_HybridSignatureMode(t *testing.T) {
	params := DefaultParams()
	if params.HybridSignatureMode != HybridOptional {
		t.Errorf("default HybridSignatureMode should be HybridOptional (1), got %d", params.HybridSignatureMode)
	}
}

func TestGenesisState_ValidateHybridMode(t *testing.T) {
	// Valid mode
	gs := DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Errorf("default genesis should be valid: %v", err)
	}

	// All valid modes
	for _, mode := range []HybridSignatureMode{HybridDisabled, HybridOptional, HybridRequired} {
		gs := DefaultGenesisState()
		gs.Params.HybridSignatureMode = mode
		if err := gs.Validate(); err != nil {
			t.Errorf("mode %d should be valid: %v", mode, err)
		}
	}

	// Invalid mode
	gs2 := DefaultGenesisState()
	gs2.Params.HybridSignatureMode = HybridSignatureMode(5)
	if err := gs2.Validate(); err == nil {
		t.Error("mode 5 should fail genesis validation")
	}
}
