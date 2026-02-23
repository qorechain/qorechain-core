package types

import (
	"testing"
)

func TestAlgorithmIDString(t *testing.T) {
	tests := []struct {
		id       AlgorithmID
		expected string
	}{
		{AlgorithmUnspecified, "unspecified"},
		{AlgorithmDilithium5, "dilithium5"},
		{AlgorithmMLKEM1024, "mlkem1024"},
		{AlgorithmID(99), "algorithm_99"},
	}

	for _, tt := range tests {
		got := tt.id.String()
		if got != tt.expected {
			t.Errorf("AlgorithmID(%d).String() = %q, want %q", tt.id, got, tt.expected)
		}
	}
}

func TestAlgorithmIDFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  AlgorithmID
		wantErr bool
	}{
		{"dilithium5 lowercase", "dilithium5", AlgorithmDilithium5, false},
		{"DILITHIUM5 uppercase", "DILITHIUM5", AlgorithmDilithium5, false},
		{"mlkem1024 lowercase", "mlkem1024", AlgorithmMLKEM1024, false},
		{"MLKEM1024 uppercase", "MLKEM1024", AlgorithmMLKEM1024, false},
		{"unknown algorithm", "unknown", AlgorithmUnspecified, true},
		{"empty string", "", AlgorithmUnspecified, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AlgorithmIDFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("AlgorithmIDFromString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.wantID {
				t.Errorf("AlgorithmIDFromString(%q) = %d, want %d", tt.input, got, tt.wantID)
			}
		})
	}
}

func TestAlgorithmIDIsSignature(t *testing.T) {
	if !AlgorithmDilithium5.IsSignature() {
		t.Error("AlgorithmDilithium5 should be a signature algorithm")
	}
	if AlgorithmMLKEM1024.IsSignature() {
		t.Error("AlgorithmMLKEM1024 should not be a signature algorithm")
	}
	if AlgorithmUnspecified.IsSignature() {
		t.Error("AlgorithmUnspecified should not be a signature algorithm")
	}
}

func TestAlgorithmIDIsKEM(t *testing.T) {
	if !AlgorithmMLKEM1024.IsKEM() {
		t.Error("AlgorithmMLKEM1024 should be a KEM algorithm")
	}
	if AlgorithmDilithium5.IsKEM() {
		t.Error("AlgorithmDilithium5 should not be a KEM algorithm")
	}
	if AlgorithmUnspecified.IsKEM() {
		t.Error("AlgorithmUnspecified should not be a KEM algorithm")
	}
}

func TestAlgorithmStatusString(t *testing.T) {
	tests := []struct {
		status   AlgorithmStatus
		expected string
	}{
		{StatusActive, "active"},
		{StatusMigrating, "migrating"},
		{StatusDeprecated, "deprecated"},
		{StatusDisabled, "disabled"},
		{AlgorithmStatus(99), "status_99"},
	}

	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.expected {
			t.Errorf("AlgorithmStatus(%d).String() = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

func TestAlgorithmInfoValidate(t *testing.T) {
	tests := []struct {
		name    string
		info    AlgorithmInfo
		wantErr bool
	}{
		{
			"valid dilithium5",
			DefaultDilithium5Info(),
			false,
		},
		{
			"valid mlkem1024",
			DefaultMLKEM1024Info(),
			false,
		},
		{
			"unspecified ID",
			AlgorithmInfo{ID: AlgorithmUnspecified, Name: "test", Category: CategorySignature, NISTLevel: 5, PublicKeySize: 100, PrivateKeySize: 100},
			true,
		},
		{
			"empty name",
			AlgorithmInfo{ID: AlgorithmID(3), Name: "", Category: CategorySignature, NISTLevel: 5, PublicKeySize: 100, PrivateKeySize: 100},
			true,
		},
		{
			"invalid category",
			AlgorithmInfo{ID: AlgorithmID(3), Name: "test", Category: "invalid", NISTLevel: 5, PublicKeySize: 100, PrivateKeySize: 100},
			true,
		},
		{
			"nist level 0",
			AlgorithmInfo{ID: AlgorithmID(3), Name: "test", Category: CategorySignature, NISTLevel: 0, PublicKeySize: 100, PrivateKeySize: 100},
			true,
		},
		{
			"nist level 6",
			AlgorithmInfo{ID: AlgorithmID(3), Name: "test", Category: CategorySignature, NISTLevel: 6, PublicKeySize: 100, PrivateKeySize: 100},
			true,
		},
		{
			"zero pubkey size",
			AlgorithmInfo{ID: AlgorithmID(3), Name: "test", Category: CategorySignature, NISTLevel: 5, PublicKeySize: 0, PrivateKeySize: 100},
			true,
		},
		{
			"zero privkey size",
			AlgorithmInfo{ID: AlgorithmID(3), Name: "test", Category: CategorySignature, NISTLevel: 5, PublicKeySize: 100, PrivateKeySize: 0},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.info.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AlgorithmInfo.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultDilithium5Info(t *testing.T) {
	info := DefaultDilithium5Info()

	if info.ID != AlgorithmDilithium5 {
		t.Errorf("expected ID %d, got %d", AlgorithmDilithium5, info.ID)
	}
	if info.Name != "dilithium5" {
		t.Errorf("expected name dilithium5, got %s", info.Name)
	}
	if info.Category != CategorySignature {
		t.Errorf("expected category %s, got %s", CategorySignature, info.Category)
	}
	if info.NISTLevel != 5 {
		t.Errorf("expected NIST level 5, got %d", info.NISTLevel)
	}
	if info.PublicKeySize != 2592 {
		t.Errorf("expected pubkey size 2592, got %d", info.PublicKeySize)
	}
	if info.PrivateKeySize != 4896 {
		t.Errorf("expected privkey size 4896, got %d", info.PrivateKeySize)
	}
	if info.SignatureSize != 4627 {
		t.Errorf("expected sig size 4627, got %d", info.SignatureSize)
	}
	if info.Status != StatusActive {
		t.Errorf("expected status active, got %s", info.Status)
	}
}

func TestDefaultMLKEM1024Info(t *testing.T) {
	info := DefaultMLKEM1024Info()

	if info.ID != AlgorithmMLKEM1024 {
		t.Errorf("expected ID %d, got %d", AlgorithmMLKEM1024, info.ID)
	}
	if info.Name != "mlkem1024" {
		t.Errorf("expected name mlkem1024, got %s", info.Name)
	}
	if info.Category != CategoryKEM {
		t.Errorf("expected category %s, got %s", CategoryKEM, info.Category)
	}
	if info.PublicKeySize != 1568 {
		t.Errorf("expected pubkey size 1568, got %d", info.PublicKeySize)
	}
	if info.PrivateKeySize != 3168 {
		t.Errorf("expected privkey size 3168, got %d", info.PrivateKeySize)
	}
	if info.CiphertextSize != 1568 {
		t.Errorf("expected ciphertext size 1568, got %d", info.CiphertextSize)
	}
}
